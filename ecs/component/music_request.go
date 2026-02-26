package component

// MusicRequest is a one-shot request for global music playback.
//
// The music system guarantees only one active song at a time. When a new
// request arrives while another song is active, the current song fades out to
// silence, then the requested song starts immediately.
type MusicRequest struct {
	Track         string
	Volume        float64
	Loop          bool
	FadeOutFrames int
}

var MusicRequestComponent = NewComponent[MusicRequest]()
