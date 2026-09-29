package domain

import (
	"testing"
)

func TestTimelineDomain_TimeConversions(t *testing.T) {
	micros := SecondsToMicroseconds(2.5)
	if micros != 2_500_000 {
		t.Errorf("expected 2500000 micros, got %d", micros)
	}

	secs := MicrosecondsToSeconds(2_500_000)
	if secs != 2.5 {
		t.Errorf("expected 2.5 secs, got %f", secs)
	}
}

func TestTimelineDomain_ClipInvariants(t *testing.T) {
	videoTrack := &Track{ID: "tr-1", Type: TrackTypeVideo}
	audioTrack := &Track{ID: "tr-2", Type: TrackTypeAudio}

	videoMedia := &MediaAsset{
		ID:       "m-1",
		MimeType: "video/mp4",
		Duration: 10.0, // 10,000,000 micros
	}

	audioMedia := &MediaAsset{
		ID:       "m-2",
		MimeType: "audio/wav",
		Duration: 5.0,
	}

	t.Run("valid clip passes", func(t *testing.T) {
		err := ValidateClipInvariants(videoTrack, videoMedia, 0, 0, 5_000_000)
		if err != nil {
			t.Errorf("expected valid clip to pass, got %v", err)
		}
	})

	t.Run("negative timeline_start rejected", func(t *testing.T) {
		err := ValidateClipInvariants(videoTrack, videoMedia, -1000, 0, 5_000_000)
		if err != ErrInvalidTimelinePosition {
			t.Errorf("expected ErrInvalidTimelinePosition, got %v", err)
		}
	})

	t.Run("negative source_in rejected", func(t *testing.T) {
		err := ValidateClipInvariants(videoTrack, videoMedia, 0, -100, 5_000_000)
		if err != ErrInvalidSourceRange {
			t.Errorf("expected ErrInvalidSourceRange, got %v", err)
		}
	})

	t.Run("source_out equal or less than source_in rejected", func(t *testing.T) {
		err := ValidateClipInvariants(videoTrack, videoMedia, 0, 5_000_000, 5_000_000)
		if err != ErrInvalidSourceRange {
			t.Errorf("expected ErrInvalidSourceRange, got %v", err)
		}

		err = ValidateClipInvariants(videoTrack, videoMedia, 0, 5_000_000, 2_000_000)
		if err != ErrInvalidSourceRange {
			t.Errorf("expected ErrInvalidSourceRange, got %v", err)
		}
	})

	t.Run("source_out exceeds media duration rejected", func(t *testing.T) {
		err := ValidateClipInvariants(videoTrack, videoMedia, 0, 0, 15_000_000) // 15s > 10s
		if err != ErrSourceOutExceedsMedia {
			t.Errorf("expected ErrSourceOutExceedsMedia, got %v", err)
		}
	})

	t.Run("incompatible media on video track rejected", func(t *testing.T) {
		err := ValidateClipInvariants(videoTrack, audioMedia, 0, 0, 2_000_000)
		if err != ErrIncompatibleMediaTrack {
			t.Errorf("expected ErrIncompatibleMediaTrack, got %v", err)
		}
	})

	t.Run("audio track accepts audio media", func(t *testing.T) {
		err := ValidateClipInvariants(audioTrack, audioMedia, 0, 0, 2_000_000)
		if err != nil {
			t.Errorf("expected audio track to accept audio media, got %v", err)
		}
	})
}

func TestTimelineDomain_OverlapLogic(t *testing.T) {
	// Clip A: [1_000_000, 3_000_000) (duration = 2_000_000)
	startA := int64(1_000_000)
	durA := int64(2_000_000)

	t.Run("abutting clips do not overlap", func(t *testing.T) {
		// Clip B right after A: [3_000_000, 5_000_000)
		if ClipsOverlap(startA, durA, 3_000_000, 2_000_000) {
			t.Errorf("abutting clips at 3_000_000 should not overlap")
		}

		// Clip B right before A: [0, 1_000_000)
		if ClipsOverlap(startA, durA, 0, 1_000_000) {
			t.Errorf("abutting clips at 1_000_000 should not overlap")
		}
	})

	t.Run("overlapping clips detected", func(t *testing.T) {
		// Inside: [1_500_000, 2_500_000)
		if !ClipsOverlap(startA, durA, 1_500_000, 1_000_000) {
			t.Errorf("nested clip should overlap")
		}

		// Partially overlapping left: [500_000, 1_500_000)
		if !ClipsOverlap(startA, durA, 500_000, 1_000_000) {
			t.Errorf("left-overlapping clip should overlap")
		}

		// Partially overlapping right: [2_500_000, 4_000_000)
		if !ClipsOverlap(startA, durA, 2_500_000, 1_500_000) {
			t.Errorf("right-overlapping clip should overlap")
		}

		// Exact match: [1_000_000, 3_000_000)
		if !ClipsOverlap(startA, durA, 1_000_000, 2_000_000) {
			t.Errorf("identical interval should overlap")
		}
	})

	t.Run("disjoint clips do not overlap", func(t *testing.T) {
		// Far left: [0, 500_000)
		if ClipsOverlap(startA, durA, 0, 500_000) {
			t.Errorf("disjoint clip on left should not overlap")
		}

		// Far right: [5_000_000, 7_000_000)
		if ClipsOverlap(startA, durA, 5_000_000, 2_000_000) {
			t.Errorf("disjoint clip on right should not overlap")
		}
	})
}
