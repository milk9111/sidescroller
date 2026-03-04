package module

import (
	"fmt"
	"strings"

	"github.com/d5/tengo/v2"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
)

const (
	defaultMusicFadeFrames = 30
)

func MusicModule() Module {
	return Module{
		Name: "music",
		Build: func(world *ecs.World, byGameEntityID map[string]ecs.Entity, owner, target ecs.Entity) map[string]tengo.Object {
			values := map[string]tengo.Object{}

			values["play"] = &tengo.UserFunction{Name: "play", Value: func(args ...tengo.Object) (tengo.Object, error) {
				if len(args) < 1 {
					return tengo.FalseValue, fmt.Errorf("play requires at least 1 argument: the name of the music track to play")
				}

				name := strings.TrimSpace(objectAsString(args[0]))
				if name == "" {
					return tengo.FalseValue, fmt.Errorf("invalid music track name")
				}

				RequestMusic(world, name)

				return tengo.TrueValue, nil
			}}

			values["stop"] = &tengo.UserFunction{Name: "stop", Value: func(args ...tengo.Object) (tengo.Object, error) {
				StopMusic(world)
				return tengo.TrueValue, nil
			}}

			return values
		},
	}
}

func RequestMusic(w *ecs.World, track string) {
	RequestMusicWithOptions(w, &component.MusicRequest{Track: track, Volume: 0, Loop: true, FadeOutFrames: defaultMusicFadeFrames})
}

func RequestMusicWithOptions(w *ecs.World, req *component.MusicRequest) {
	if w == nil || req == nil {
		return
	}
	ent := ecs.CreateEntity(w)
	_ = ecs.Add(w, ent, component.MusicRequestComponent.Kind(), req)
}

func StopMusic(w *ecs.World) {
	RequestMusicWithOptions(w, &component.MusicRequest{FadeOutFrames: defaultMusicFadeFrames})
}
