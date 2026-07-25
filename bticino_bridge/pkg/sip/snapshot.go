// snapshot.go implements on-demand JPEG snapshot capture from the device's
// Linux framebuffer (/dev/fb0). This approach works on the Classe 100X where
// the *7*300 RTP duplication mechanism (used on the Classe 300X) does not
// produce decodable video on local UDP ports.
//
// When the camera is active (doorbell ring, "eye" button, or active call),
// the framebuffer contains the live camera feed displayed on the LCD screen.
// We simply read the raw BGRA pixels and encode them as JPEG.
package sip

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"io"
	"os"
	"sync"
	"time"

	"bticino_bridge/pkg/openwebnet"

	"github.com/sirupsen/logrus"
)

const (
	// Framebuffer parameters for Classe 100X (800x480 LCD, 32-bit BGRA)
	fbDevice = "/dev/fb0"
	fbWidth  = 800
	fbHeight = 480
	fbBPP    = 4    // bytes per pixel (32-bit BGRA)
	fbStride = 3200 // fbWidth * fbBPP
	fbSize   = fbStride * fbHeight // 1,536,000 bytes
)

// SnapshotService captures JPEG snapshots from the device framebuffer.
type SnapshotService struct {
	own    *openwebnet.Client // retained for API compatibility
	logger *logrus.Logger

	mu       sync.Mutex
	lastJPEG []byte
	lastTime time.Time
}

// NewSnapshotService creates the snapshot service.
func NewSnapshotService(own *openwebnet.Client, logger *logrus.Logger) *SnapshotService {
	if logger == nil {
		logger = logrus.New()
	}
	return &SnapshotService{
		own:    own,
		logger: logger,
	}
}

// Capture returns a JPEG snapshot from the device framebuffer (/dev/fb0).
// If a cached capture exists within maxAge, it is reused. The timeout
// parameter is retained for API compatibility but not used in fb capture.
func (s *SnapshotService) Capture(timeout, maxAge time.Duration) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if maxAge > 0 && s.lastJPEG != nil && time.Since(s.lastTime) < maxAge {
		s.logger.Debugf("Snapshot: returning cached capture (%s old)", time.Since(s.lastTime).Round(time.Millisecond))
		return s.lastJPEG, nil
	}

	// 1) Read raw framebuffer
	f, err := os.Open(fbDevice)
	if err != nil {
		return nil, fmt.Errorf("cannot open framebuffer %s: %w", fbDevice, err)
	}
	defer f.Close()

	raw := make([]byte, fbSize)
	if _, err := io.ReadFull(f, raw); err != nil {
		return nil, fmt.Errorf("cannot read framebuffer: %w", err)
	}

	// 2) Convert BGRA framebuffer to RGBA image
	img := image.NewNRGBA(image.Rect(0, 0, fbWidth, fbHeight))
	for y := 0; y < fbHeight; y++ {
		srcOff := y * fbStride
		dstOff := y * img.Stride
		for x := 0; x < fbWidth; x++ {
			img.Pix[dstOff+0] = raw[srcOff+2] // R (from BGRA offset 2)
			img.Pix[dstOff+1] = raw[srcOff+1] // G (from BGRA offset 1)
			img.Pix[dstOff+2] = raw[srcOff+0] // B (from BGRA offset 0)
			img.Pix[dstOff+3] = 255            // A (fully opaque)
			srcOff += fbBPP
			dstOff += 4
		}
	}

	// 3) Encode as JPEG
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 90}); err != nil {
		return nil, fmt.Errorf("JPEG encoding failed: %w", err)
	}

	jpegData := buf.Bytes()
	s.lastJPEG = jpegData
	s.lastTime = time.Now()
	s.logger.Infof("Snapshot captured from framebuffer: %d bytes (%dx%d)", len(jpegData), fbWidth, fbHeight)
	return jpegData, nil
}

// LastSnapshot returns the most recent capture and its timestamp (nil if none).
func (s *SnapshotService) LastSnapshot() ([]byte, time.Time) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastJPEG, s.lastTime
}
