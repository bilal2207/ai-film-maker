package media

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/ai-filmmaker/api/internal/domain"
)

type Processor interface {
	InspectMetadata(ctx context.Context, inputPath string) (*domain.MediaMetadata, error)
	GenerateProxy(ctx context.Context, inputPath, outputPath string) error
	GenerateThumbnail(ctx context.Context, inputPath, outputPath string, atSeconds float64) error
}

type FFmpegProcessor struct {
	ffmpegPath  string
	ffprobePath string
}

func NewFFmpegProcessor(ffmpegPath, ffprobePath string) (*FFmpegProcessor, error) {
	resolvedFFmpeg := resolveBinary(ffmpegPath, "ffmpeg")
	resolvedFFprobe := resolveBinary(ffprobePath, "ffprobe")

	if resolvedFFmpeg == "" {
		return nil, errors.New("ffmpeg executable not found in PATH or configured paths")
	}
	if resolvedFFprobe == "" {
		return nil, errors.New("ffprobe executable not found in PATH or configured paths")
	}

	return &FFmpegProcessor{
		ffmpegPath:  resolvedFFmpeg,
		ffprobePath: resolvedFFprobe,
	}, nil
}

func resolveBinary(customPath, name string) string {
	if customPath != "" {
		if _, err := exec.LookPath(customPath); err == nil {
			return customPath
		}
		if _, err := os.Stat(customPath); err == nil {
			return customPath
		}
	}

	// 1. Check system PATH
	if p, err := exec.LookPath(name); err == nil {
		return p
	}

	// 2. Check standard local tools directories
	homeDir, _ := os.UserHomeDir()
	candidates := []string{
		filepath.Join(homeDir, ".tools", "ffmpeg", "bin", name+".exe"),
		filepath.Join(homeDir, ".tools", "ffmpeg", name+".exe"),
		"C:\\Program Files\\ffmpeg\\bin\\" + name + ".exe",
		"/usr/bin/" + name,
		"/usr/local/bin/" + name,
	}

	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return ""
}

type ffprobeOutput struct {
	Streams []struct {
		CodecType  string `json:"codec_type"`
		CodecName  string `json:"codec_name"`
		Width      int    `json:"width"`
		Height     int    `json:"height"`
		RFrameRate string `json:"r_frame_rate"`
		Duration   string `json:"duration"`
	} `json:"streams"`
	Format struct {
		Duration string `json:"duration"`
		Size     string `json:"size"`
	} `json:"format"`
}

func (p *FFmpegProcessor) InspectMetadata(ctx context.Context, inputPath string) (*domain.MediaMetadata, error) {
	args := []string{
		"-v", "error",
		"-show_entries", "format=duration,size:stream=width,height,r_frame_rate,codec_name,codec_type,duration",
		"-of", "json",
		inputPath,
	}

	cmd := exec.CommandContext(ctx, p.ffprobePath, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffprobe inspection failed: %w (stderr: %s)", err, stderr.String())
	}

	var probe ffprobeOutput
	if err := json.Unmarshal(stdout.Bytes(), &probe); err != nil {
		return nil, fmt.Errorf("failed to parse ffprobe json: %w", err)
	}

	meta := &domain.MediaMetadata{}

	// Extract format-level duration
	if probe.Format.Duration != "" {
		if d, err := strconv.ParseFloat(probe.Format.Duration, 64); err == nil {
			meta.Duration = d
		}
	}

	// Extract first video stream metadata
	for _, stream := range probe.Streams {
		if stream.CodecType == "video" {
			meta.Width = stream.Width
			meta.Height = stream.Height
			meta.Codec = stream.CodecName

			if meta.Duration == 0 && stream.Duration != "" {
				if d, err := strconv.ParseFloat(stream.Duration, 64); err == nil {
					meta.Duration = d
				}
			}

			if stream.RFrameRate != "" {
				meta.FPS = parseFrameRate(stream.RFrameRate)
			}
			break
		}
	}

	return meta, nil
}

func parseFrameRate(rFrameRate string) float64 {
	parts := strings.Split(rFrameRate, "/")
	if len(parts) == 1 {
		f, _ := strconv.ParseFloat(parts[0], 64)
		return f
	}
	if len(parts) == 2 {
		num, err1 := strconv.ParseFloat(parts[0], 64)
		den, err2 := strconv.ParseFloat(parts[1], 64)
		if err1 == nil && err2 == nil && den > 0 {
			return num / den
		}
	}
	return 0
}

func (p *FFmpegProcessor) GenerateProxy(ctx context.Context, inputPath, outputPath string) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	args := []string{
		"-y",
		"-i", inputPath,
		"-vf", "scale='min(1280,iw)':-2",
		"-c:v", "libx264",
		"-preset", "fast",
		"-crf", "23",
		"-c:a", "aac",
		"-b:a", "128k",
		"-movflags", "+faststart",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, p.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg proxy transcode failed: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}

func (p *FFmpegProcessor) GenerateThumbnail(ctx context.Context, inputPath, outputPath string, atSeconds float64) error {
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return err
	}

	if atSeconds < 0 {
		atSeconds = 0
	}
	timeStr := fmt.Sprintf("%.2f", atSeconds)

	args := []string{
		"-y",
		"-ss", timeStr,
		"-i", inputPath,
		"-vframes", "1",
		"-q:v", "2",
		outputPath,
	}

	cmd := exec.CommandContext(ctx, p.ffmpegPath, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("ffmpeg thumbnail extraction failed: %w (stderr: %s)", err, stderr.String())
	}
	return nil
}
