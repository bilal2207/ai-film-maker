package domain

import (
	"errors"
	"path/filepath"
	"strings"
	"time"
)

type MediaStatus string

const (
	StatusUploading  MediaStatus = "UPLOADING"
	StatusProcessing MediaStatus = "PROCESSING"
	StatusReady      MediaStatus = "READY"
	StatusFailed     MediaStatus = "FAILED"
)

var (
	ErrMediaNotFound           = errors.New("media asset not found")
	ErrInvalidMediaID          = errors.New("invalid media id")
	ErrInvalidFilename         = errors.New("invalid original filename")
	ErrUnsupportedMimeType     = errors.New("unsupported video mime type")
	ErrFileSizeExceeded        = errors.New("file size exceeds maximum allowed limit (5GB)")
	ErrInvalidStatusTransition = errors.New("invalid media status transition")
	ErrMediaNotBelongToProject = errors.New("media asset does not belong to the specified project")
	ErrObjectNotFoundInStorage = errors.New("uploaded media object not found in storage")
)

// AllowedVideoMimeTypes defines supported video container formats for ingestion
var AllowedVideoMimeTypes = map[string]bool{
	"video/mp4":                true,
	"video/quicktime":          true,
	"video/webm":               true,
	"video/x-matroska":         true,
	"video/x-msvideo":          true,
	"video/mpeg":               true,
	"application/octet-stream": true, // some clients upload raw video as octet-stream; verified later by ffprobe
}

const MaxAllowedFileSizeBytes = 5 * 1024 * 1024 * 1024 // 5 GB

type MediaAsset struct {
	ID                 string      `json:"id"`
	ProjectID          string      `json:"projectId"`
	OriginalObjectKey  string      `json:"originalObjectKey"`
	ProxyObjectKey     *string     `json:"proxyObjectKey,omitempty"`
	ThumbnailObjectKey *string     `json:"thumbnailObjectKey,omitempty"`
	OriginalFilename   string      `json:"originalFilename"`
	MimeType           string      `json:"mimeType"`
	FileSize           int64       `json:"fileSize"`
	Duration           float64     `json:"duration"`
	Width              int         `json:"width"`
	Height             int         `json:"height"`
	FPS                float64     `json:"fps"`
	Status             MediaStatus `json:"status"`
	ErrorMessage       *string     `json:"errorMessage,omitempty"`
	CreatedAt          time.Time   `json:"createdAt"`
	UpdatedAt          time.Time   `json:"updatedAt"`

	// URLs dynamically populated on response if available
	DownloadURL  *string `json:"downloadUrl,omitempty"`
	ProxyURL     *string `json:"proxyUrl,omitempty"`
	ThumbnailURL *string `json:"thumbnailUrl,omitempty"`
}

func (m *MediaAsset) CanTransitionTo(target MediaStatus) bool {
	switch m.Status {
	case StatusUploading:
		return target == StatusProcessing || target == StatusFailed
	case StatusProcessing:
		return target == StatusReady || target == StatusFailed
	case StatusFailed:
		return target == StatusProcessing // allows retry
	case StatusReady:
		return false
	default:
		return false
	}
}

type InitUploadInput struct {
	OriginalFilename string `json:"originalFilename"`
	MimeType         string `json:"mimeType"`
	FileSize         int64  `json:"fileSize"`
}

func (in *InitUploadInput) Validate() error {
	in.OriginalFilename = strings.TrimSpace(in.OriginalFilename)
	if in.OriginalFilename == "" {
		return ErrInvalidFilename
	}

	// Basic path sanitization
	base := filepath.Base(in.OriginalFilename)
	if base == "." || base == "/" || base == "\\" {
		return ErrInvalidFilename
	}
	in.OriginalFilename = base

	in.MimeType = strings.ToLower(strings.TrimSpace(in.MimeType))
	if in.MimeType == "" || !AllowedVideoMimeTypes[in.MimeType] {
		// Fallback check extension if mime type is generic
		ext := strings.ToLower(filepath.Ext(in.OriginalFilename))
		if ext != ".mp4" && ext != ".mov" && ext != ".webm" && ext != ".mkv" && ext != ".avi" {
			return ErrUnsupportedMimeType
		}
		if in.MimeType == "" {
			in.MimeType = "video/mp4"
		}
	}

	if in.FileSize <= 0 {
		return errors.New("file size must be greater than zero bytes")
	}

	if in.FileSize > MaxAllowedFileSizeBytes {
		return ErrFileSizeExceeded
	}

	return nil
}

type InitUploadOutput struct {
	Media     *MediaAsset `json:"media"`
	UploadURL string      `json:"uploadUrl"`
	ObjectKey string      `json:"objectKey"`
}

type MediaMetadata struct {
	Duration float64
	Width    int
	Height   int
	FPS      float64
	Codec    string
}
