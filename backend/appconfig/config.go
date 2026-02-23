package appconfig

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
)

const (
	RealtimeModeStrictNoDrop = "strict_no_drop"
	RealtimeModeAllowDrop    = "allow_drop"
)

type Config struct {
	StorageRoot string        `json:"storage_root"`
	HLS         HLSConfig     `json:"hls"`
	Encoder     EncoderConfig `json:"encoder"`
	Auth        AuthConfig    `json:"auth"`
	Cameras     []Camera      `json:"cameras"`
}

type HLSConfig struct {
	SegmentDurationSeconds int      `json:"segment_duration_seconds"`
	PlaylistSize           int      `json:"playlist_size"`
	DeleteOldSegments      bool     `json:"delete_old_segments"`
	Flags                  []string `json:"flags"`
	SegmentType            string   `json:"segment_type"`
	SegmentFilenamePattern string   `json:"segment_filename_pattern"`
	PlaylistFilename       string   `json:"playlist_filename"`
}

type EncoderConfig struct {
	Codec       string            `json:"codec"`
	Preset      int               `json:"preset"`
	CRF         int               `json:"crf"`
	GOPSeconds  int               `json:"gop_seconds"`
	Threads     int               `json:"threads"`
	PixelFormat string            `json:"pixel_format"`
	FrameRate   int               `json:"frame_rate"`
	MaxBFrames  int               `json:"max_b_frames"`
	Options     map[string]string `json:"options"`
	Realtime    RealtimeConfig    `json:"realtime"`
}

type RealtimeConfig struct {
	Mode         string `json:"mode"`
	LagWarnMS    int    `json:"lag_warn_ms"`
	LagRecoverMS int    `json:"lag_recover_ms"`
	AutoTune     bool   `json:"auto_tune"`
	QueueSize    int    `json:"queue_size"`
}

type AuthConfig struct {
	MiddlewareEnabled bool `json:"middleware_enabled"`
}

type Camera struct {
	Name    string `json:"name"`
	RTSPURL string `json:"rtsp_url"`
	Enabled bool   `json:"enabled"`
}

func LoadConfig(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config file failed: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(b, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config json failed: %w", err)
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func DefaultConfig() Config {
	return Config{
		StorageRoot: "storage",
		HLS: HLSConfig{
			SegmentDurationSeconds: 60,
			PlaylistSize:           0,
			DeleteOldSegments:      false,
			Flags: []string{
				"independent_segments",
				"program_date_time",
			},
			SegmentType:            "fmp4",
			SegmentFilenamePattern: "seg_%06d.m4s",
			PlaylistFilename:       "index.m3u8",
		},
		Encoder: EncoderConfig{
			Codec:       "libsvtav1",
			Preset:      13,
			CRF:         48,
			GOPSeconds:  2,
			Threads:     0,
			PixelFormat: "yuv420p",
			FrameRate:   25,
			MaxBFrames:  0,
			Options: map[string]string{
				"tune": "0",
			},
			Realtime: RealtimeConfig{
				Mode:         RealtimeModeAllowDrop,
				LagWarnMS:    500,
				LagRecoverMS: 2000,
				AutoTune:     true,
				QueueSize:    64,
			},
		},
		Auth: AuthConfig{
			MiddlewareEnabled: false,
		},
		Cameras: DefaultCameras(),
	}
}

func DefaultCameras() []Camera {
	cams := make([]Camera, 0, 8)
	for i := 1; i <= 8; i++ {
		cams = append(cams, Camera{
			Name:    fmt.Sprintf("cam%d", i),
			RTSPURL: fmt.Sprintf("rtsp://localhost:8554/cam%d", i),
			Enabled: i == 1,
		})
	}
	return cams
}

func (c *Config) applyDefaults() {
	d := DefaultConfig()
	if strings.TrimSpace(c.StorageRoot) == "" {
		c.StorageRoot = d.StorageRoot
	}

	if c.HLS.SegmentDurationSeconds <= 0 {
		c.HLS.SegmentDurationSeconds = d.HLS.SegmentDurationSeconds
	}
	if c.HLS.PlaylistSize < 0 {
		c.HLS.PlaylistSize = d.HLS.PlaylistSize
	}
	if len(c.HLS.Flags) == 0 {
		c.HLS.Flags = append([]string(nil), d.HLS.Flags...)
	}
	if strings.TrimSpace(c.HLS.SegmentType) == "" {
		c.HLS.SegmentType = d.HLS.SegmentType
	}
	if strings.TrimSpace(c.HLS.SegmentFilenamePattern) == "" {
		c.HLS.SegmentFilenamePattern = d.HLS.SegmentFilenamePattern
	}
	if strings.TrimSpace(c.HLS.PlaylistFilename) == "" {
		c.HLS.PlaylistFilename = d.HLS.PlaylistFilename
	}

	if strings.TrimSpace(c.Encoder.Codec) == "" {
		c.Encoder.Codec = d.Encoder.Codec
	}
	if c.Encoder.Preset <= 0 {
		c.Encoder.Preset = d.Encoder.Preset
	}
	if c.Encoder.CRF <= 0 {
		c.Encoder.CRF = d.Encoder.CRF
	}
	if c.Encoder.GOPSeconds <= 0 {
		c.Encoder.GOPSeconds = d.Encoder.GOPSeconds
	}
	if strings.TrimSpace(c.Encoder.PixelFormat) == "" {
		c.Encoder.PixelFormat = d.Encoder.PixelFormat
	}
	if c.Encoder.FrameRate <= 0 {
		c.Encoder.FrameRate = d.Encoder.FrameRate
	}
	if c.Encoder.Options == nil {
		c.Encoder.Options = map[string]string{}
	}
	if c.Encoder.MaxBFrames < 0 {
		c.Encoder.MaxBFrames = d.Encoder.MaxBFrames
	}

	if strings.TrimSpace(c.Encoder.Realtime.Mode) == "" {
		c.Encoder.Realtime.Mode = d.Encoder.Realtime.Mode
	}
	if c.Encoder.Realtime.LagWarnMS <= 0 {
		c.Encoder.Realtime.LagWarnMS = d.Encoder.Realtime.LagWarnMS
	}
	if c.Encoder.Realtime.LagRecoverMS <= 0 {
		c.Encoder.Realtime.LagRecoverMS = d.Encoder.Realtime.LagRecoverMS
	}
	if c.Encoder.Realtime.QueueSize <= 0 {
		c.Encoder.Realtime.QueueSize = d.Encoder.Realtime.QueueSize
	}

	if len(c.Cameras) == 0 {
		c.Cameras = d.Cameras
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.StorageRoot) == "" {
		return errors.New("storage_root must not be empty")
	}
	if c.HLS.SegmentDurationSeconds <= 0 {
		return errors.New("hls.segment_duration_seconds must be > 0")
	}
	if strings.TrimSpace(c.HLS.SegmentFilenamePattern) == "" {
		return errors.New("hls.segment_filename_pattern must not be empty")
	}
	if strings.TrimSpace(c.HLS.PlaylistFilename) == "" {
		return errors.New("hls.playlist_filename must not be empty")
	}
	if strings.TrimSpace(c.Encoder.Codec) == "" {
		return errors.New("encoder.codec must not be empty")
	}
	if c.Encoder.Realtime.Mode != RealtimeModeStrictNoDrop && c.Encoder.Realtime.Mode != RealtimeModeAllowDrop {
		return fmt.Errorf("encoder.realtime.mode must be one of %q or %q", RealtimeModeStrictNoDrop, RealtimeModeAllowDrop)
	}
	if c.Encoder.Realtime.QueueSize <= 0 {
		return errors.New("encoder.realtime.queue_size must be > 0")
	}

	seen := make(map[string]struct{}, len(c.Cameras))
	enabled := 0
	for _, cam := range c.Cameras {
		name := strings.TrimSpace(cam.Name)
		url := strings.TrimSpace(cam.RTSPURL)
		if name == "" {
			return errors.New("camera.name must not be empty")
		}
		if url == "" {
			return fmt.Errorf("camera %q rtsp_url must not be empty", name)
		}
		if _, ok := seen[name]; ok {
			return fmt.Errorf("camera %q is duplicated", name)
		}
		seen[name] = struct{}{}
		if cam.Enabled {
			enabled++
		}
	}
	if enabled == 0 {
		return errors.New("at least one camera must be enabled")
	}
	return nil
}
