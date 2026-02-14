package assets

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	_ "image/png"
	"log"
	"path/filepath"
	"strings"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/audio"
	"github.com/hajimehoshi/ebiten/v2/audio/wav"
)

//go:embed *
var assetsFS embed.FS

// PlayerTemplateSheet is the embedded player sprite sheet as an *ebiten.Image.
var PlayerTemplateSheet *ebiten.Image
var PlayerSheet *ebiten.Image
var PlayerV2Sheet *ebiten.Image
var AimTargetInvalid *ebiten.Image
var AimTargetValid *ebiten.Image
var Claw *ebiten.Image
var audioContext = audio.NewContext(44100)

func init() {
	PlayerTemplateSheet = loadImageFromAssets("player_template-Sheet.png")
	PlayerSheet = loadImageFromAssets("player-Sheet.png")
	PlayerV2Sheet = loadImageFromAssets("player_v2-Sheet.png")
	AimTargetInvalid = loadImageFromAssets("aim_target_invalid.png")
	AimTargetValid = loadImageFromAssets("aim_target_valid.png")
	Claw = loadImageFromAssets("claw.png")
}

// LoadImage loads an embedded asset by assets-relative path.
func LoadImage(path string) (*ebiten.Image, error) {
	clean := cleanAssetPath(path)
	b, err := assetsFS.ReadFile(clean)
	if err != nil {
		return nil, err
	}
	img, _, err := image.Decode(bytes.NewReader(b))
	if err != nil {
		return nil, err
	}
	return ebiten.NewImageFromImage(img), nil
}

// LoadFile loads an embedded asset by assets-relative path.
func LoadFile(path string) ([]byte, error) {
	clean := cleanAssetPath(path)
	return assetsFS.ReadFile(clean)
}

// LoadAudio loads an embedded audio asset by assets-relative path.
func LoadAudio(path string) ([]byte, error) {
	return LoadFile(path)
}

// LoadAudioPlayer loads an embedded audio asset and creates an audio player.
func LoadAudioPlayer(path string) (*audio.Player, error) {
	b, err := LoadAudio(path)
	if err != nil {
		return nil, err
	}

	clean := strings.ToLower(cleanAssetPath(path))
	reader := bytes.NewReader(b)

	if strings.HasSuffix(clean, ".wav") {
		stream, err := wav.DecodeWithSampleRate(audioContext.SampleRate(), reader)
		if err != nil {
			return nil, fmt.Errorf("decode wav %q: %w", path, err)
		}
		return audioContext.NewPlayer(stream)
	}

	// Fallback for already-decoded PCM assets in Ebiten's native format.
	return audioContext.NewPlayerFromBytes(b), nil
}

func loadImageFromAssets(path string) *ebiten.Image {
	img, err := LoadImage(path)
	if err != nil {
		log.Fatalf("embed: load %s: %v", path, err)
	}
	return img
}

func cleanAssetPath(path string) string {
	if path == "" {
		return ""
	}
	if filepath.IsAbs(path) {
		s := filepath.ToSlash(path)
		if idx := strings.LastIndex(s, "/assets/"); idx >= 0 {
			return s[idx+len("/assets/"):]
		}
		return filepath.Base(path)
	}
	s := filepath.ToSlash(path)
	if strings.HasPrefix(s, "assets/") {
		return strings.TrimPrefix(s, "assets/")
	}
	return s
}
