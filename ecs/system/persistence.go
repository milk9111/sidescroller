package system

import (
	"fmt"
	"path/filepath"

	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/ecs/entity"
	"github.com/milk9111/sidescroller/levels"
)

type PersistenceMode int

const (
	PersistenceOnLevelChange PersistenceMode = iota
	PersistenceOnReload
)

type PersistenceSystem struct {
	levelName        string
	initialLevelName string
	allAbilities     bool
	physicsReset     func()
	initialized      bool
	loadSequence     uint64
}

func NewPersistenceSystem(initialLevelName string, allAbilities bool, physicsReset func()) *PersistenceSystem {
	return &PersistenceSystem{
		levelName:        initialLevelName,
		initialLevelName: initialLevelName,
		allAbilities:     allAbilities,
		physicsReset:     physicsReset,
	}
}

func (p *PersistenceSystem) Update(w *ecs.World) {
	if p == nil || w == nil {
		return
	}

	if !p.initialized {
		if err := p.reloadWorld(w, PersistenceOnReload); err != nil {
			panic("persistence system: initial load failed: " + err.Error())
		}
		p.initialized = true
		return
	}

	if _, ok := ecs.First(w, component.ResetToInitialLevelRequestComponent.Kind()); ok {
		p.levelName = p.initialLevelName
		ecs.ForEach(w, component.ResetToInitialLevelRequestComponent.Kind(), func(e ecs.Entity, _ *component.ResetToInitialLevelRequest) {
			ecs.DestroyEntity(w, e)
		})
		if err := p.reloadWorld(w, PersistenceOnReload); err != nil {
			panic("persistence system: reset-to-initial failed: " + err.Error())
		}
		return
	}

	if _, ok := ecs.First(w, component.ReloadRequestComponent.Kind()); ok {
		ecs.ForEach(w, component.ReloadRequestComponent.Kind(), func(e ecs.Entity, _ *component.ReloadRequest) {
			ecs.DestroyEntity(w, e)
		})
		if err := p.reloadWorld(w, PersistenceOnReload); err != nil {
			panic("persistence system: reload failed: " + err.Error())
		}
		return
	}

	if req, ok := p.firstLevelChangeRequest(w); ok {
		ecs.ForEach(w, component.LevelChangeRequestComponent.Kind(), func(e ecs.Entity, _ *component.LevelChangeRequest) {
			ecs.DestroyEntity(w, e)
		})

		if req.TargetLevel != "" {
			p.levelName = req.TargetLevel
		}

		if err := p.reloadWorld(w, PersistenceOnLevelChange); err != nil {
			panic("persistence system: level transition reload failed: " + err.Error())
		}

		p.spawnPlayerAtLinkedTransition(w, req.SpawnTransitionID)
		p.applyTransitionPop(w, req)

		rtEnt := ecs.CreateEntity(w)
		_ = ecs.Add(w, rtEnt, component.TransitionRuntimeComponent.Kind(), &component.TransitionRuntime{
			Phase:   component.TransitionFadeIn,
			Alpha:   1,
			Timer:   30,
			Req:     component.LevelChangeRequest{},
			ReqSent: true,
		})
	}
}

func (p *PersistenceSystem) snapshotPersistentSingletons(w *ecs.World, mode PersistenceMode) map[string]ecs.Entity {
	preferred := map[string]ecs.Entity{}
	if w == nil {
		return preferred
	}

	ecs.ForEach(w, component.PersistentComponent.Kind(), func(e ecs.Entity, persistent *component.Persistent) {
		if persistent == nil || persistent.ID == "" || !p.shouldKeep(persistent, mode) {
			return
		}
		if _, exists := preferred[persistent.ID]; !exists {
			preferred[persistent.ID] = e
		}
	})

	return preferred
}

func (p *PersistenceSystem) pruneForReload(w *ecs.World, mode PersistenceMode) {
	if w == nil {
		return
	}

	toDestroy := make([]ecs.Entity, 0)
	for _, e := range ecs.Entities(w) {
		persistent, ok := ecs.Get(w, e, component.PersistentComponent.Kind())
		if !ok || persistent == nil || !p.shouldKeep(persistent, mode) {
			toDestroy = append(toDestroy, e)
		}
	}

	for _, e := range toDestroy {
		ecs.DestroyEntity(w, e)
	}
}

func (p *PersistenceSystem) resolvePersistentSingletons(w *ecs.World, preferred map[string]ecs.Entity) {
	if w == nil {
		return
	}

	seen := make(map[string]ecs.Entity)
	toDestroy := make([]ecs.Entity, 0)
	ecs.ForEach(w, component.PersistentComponent.Kind(), func(e ecs.Entity, persistent *component.Persistent) {
		if persistent == nil || persistent.ID == "" {
			return
		}
		if preferred != nil {
			if preferredEntity, ok := preferred[persistent.ID]; ok {
				seen[persistent.ID] = preferredEntity
				if e != preferredEntity {
					toDestroy = append(toDestroy, e)
				}
				return
			}
		}

		if existing, ok := seen[persistent.ID]; ok && existing != e {
			toDestroy = append(toDestroy, e)
			return
		}
		seen[persistent.ID] = e
	})

	for _, e := range toDestroy {
		ecs.DestroyEntity(w, e)
	}
}

func (p *PersistenceSystem) firstLevelChangeRequest(w *ecs.World) (component.LevelChangeRequest, bool) {
	ent, ok := ecs.First(w, component.LevelChangeRequestComponent.Kind())
	if !ok {
		return component.LevelChangeRequest{}, false
	}
	req, ok := ecs.Get(w, ent, component.LevelChangeRequestComponent.Kind())
	if !ok || req == nil {
		return component.LevelChangeRequest{}, false
	}
	return *req, true
}

func (p *PersistenceSystem) reloadWorld(w *ecs.World, mode PersistenceMode) error {
	preferredSingletons := p.snapshotPersistentSingletons(w, mode)
	p.pruneForReload(w, mode)

	if p.physicsReset != nil {
		p.physicsReset()
	}

	name := p.levelName
	if filepath.Ext(name) == "" {
		name += ".json"
	}

	level, err := levels.LoadLevelFromFS(name)
	if err != nil {
		return fmt.Errorf("load level %q: %w", name, err)
	}

	if err = entity.LoadLevelToWorld(w, level); err != nil {
		return err
	}

	if _, err = entity.BuildEntity(w, "camera.yaml"); err != nil {
		return err
	}

	p.resolvePersistentSingletons(w, preferredSingletons)

	if len(level.Entities) == 0 {
		if _, ok := ecs.First(w, component.PlayerTagComponent.Kind()); !ok {
			if _, err = entity.BuildEntity(w, "player.yaml"); err != nil {
				return err
			}
		}
	}

	if _, ok := ecs.First(w, component.AimTargetTagComponent.Kind()); !ok {
		if _, err = entity.BuildEntity(w, "aim_target.yaml"); err != nil {
			return err
		}
	}

	if _, ok := ecs.First(w, component.PlayerHealthBarComponent.Kind()); !ok {
		if _, err = entity.NewPlayerHealthBar(w); err != nil {
			return err
		}
	}

	if _, ok := ecs.First(w, component.TrophyCounterComponent.Kind()); !ok {
		if _, err = entity.NewTrophyCounter(w); err != nil {
			return err
		}
	}

	if _, ok := ecs.First(w, component.TrophyTrackerComponent.Kind()); !ok {
		if _, err = entity.NewTrophyTracker(w); err != nil {
			return err
		}
	}

	if _, ok := ecs.First(w, component.MusicPlayerComponent.Kind()); !ok {
		if _, err = entity.NewMusicPlayer(w); err != nil {
			return err
		}
	}

	if _, ok := ecs.First(w, component.AbilitiesComponent.Kind()); !ok {
		abEnt := ecs.CreateEntity(w)
		_ = ecs.Add(w, abEnt, component.PersistentComponent.Kind(), &component.Persistent{
			ID:                "player_abilities",
			KeepOnLevelChange: true,
			KeepOnReload:      false,
		})
		_ = ecs.Add(w, abEnt, component.AbilitiesComponent.Kind(), &component.Abilities{
			DoubleJump: p.allAbilities,
			WallGrab:   p.allAbilities,
			Anchor:     p.allAbilities,
		})
	}

	p.loadSequence++
	ent := ecs.CreateEntity(w)
	_ = ecs.Add(w, ent, component.LevelLoadedComponent.Kind(), &component.LevelLoaded{Sequence: p.loadSequence})
	return nil
}

func (p *PersistenceSystem) spawnPlayerAtLinkedTransition(w *ecs.World, transitionID string) {
	if w == nil || transitionID == "" {
		return
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}

	var (
		spawnX  float64
		spawnY  float64
		spawnH  float64
		found   bool
		isLeft  bool
		isRight bool
	)

	ecs.ForEach2(w, component.TransitionComponent.Kind(), component.TransformComponent.Kind(), func(_ ecs.Entity, tr *component.Transition, tf *component.Transform) {
		if found || tr == nil || tf == nil || tr.ID != transitionID {
			return
		}

		tw := tr.Bounds.W
		th := tr.Bounds.H
		if tw <= 0 {
			tw = 32
		}
		if th <= 0 {
			th = 32
		}

		spawnX = tf.X + tr.Bounds.X + tw/2
		spawnY = tf.Y + tr.Bounds.Y + th/2
		spawnH = th
		found = true
		isLeft = tr.EnterDir == component.TransitionDirLeft
		isRight = tr.EnterDir == component.TransitionDirRight
	})

	if !found {
		return
	}

	playerTf, ok := ecs.Get(w, player, component.TransformComponent.Kind())
	if !ok || playerTf == nil {
		playerTf = &component.Transform{ScaleX: 1, ScaleY: 1}
	}

	if playerBody, ok := ecs.Get(w, player, component.PhysicsBodyComponent.Kind()); ok && playerBody != nil && playerBody.Width > 0 && playerBody.Height > 0 {
		if !isLeft && !isRight {
			playerTf.X = spawnX - playerBody.Width/2 - playerBody.OffsetX
			playerTf.Y = spawnY - playerBody.Height/2 - playerBody.OffsetY
		} else {
			playerTf.X = spawnX - playerBody.Width/2 - playerBody.OffsetX
			playerTf.Y = spawnY + spawnH/2 - playerBody.Height
		}
	} else {
		playerTf.X = spawnX
		playerTf.Y = spawnY
	}
	_ = ecs.Add(w, player, component.TransformComponent.Kind(), playerTf)

	if playerSprite, ok := ecs.Get(w, player, component.SpriteComponent.Kind()); ok && playerSprite != nil {
		playerSprite.FacingLeft = isLeft
	}

	_ = ecs.Add(w, player, component.TransitionCooldownComponent.Kind(), &component.TransitionCooldown{Active: true, TransitionID: transitionID})
}

func (p *PersistenceSystem) applyTransitionPop(w *ecs.World, req component.LevelChangeRequest) {
	if !req.EntryFromBelow {
		return
	}

	spawnEnterDir := component.TransitionDirection("")
	ecs.ForEach2(w, component.TransitionComponent.Kind(), component.TransformComponent.Kind(), func(_ ecs.Entity, tr *component.Transition, _ *component.Transform) {
		if tr != nil && tr.ID == req.SpawnTransitionID {
			spawnEnterDir = tr.EnterDir
		}
	})
	if spawnEnterDir != component.TransitionDirUp {
		return
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}

	mv := 80.0
	jp := 120.0
	if pCfg, ok := ecs.Get(w, player, component.PlayerComponent.Kind()); ok && pCfg != nil {
		mv = pCfg.MoveSpeed
		jp = pCfg.JumpSpeed
	}

	side := 1.0
	if req.FromFacingLeft {
		side = -1.0
	}

	dur := 6
	push := mv * 8.0
	pop := &component.TransitionPop{
		VX:          side * mv * 0.75,
		VY:          -jp * 1.1,
		FacingLeft:  req.FromFacingLeft,
		WallJumpDur: dur,
		WallJumpX:   side * push,
	}
	_ = ecs.Add(w, player, component.TransitionPopComponent.Kind(), pop)
}

func (p *PersistenceSystem) shouldKeep(persistent *component.Persistent, mode PersistenceMode) bool {
	if persistent == nil {
		return false
	}
	if mode == PersistenceOnReload {
		return persistent.KeepOnReload
	}
	return persistent.KeepOnLevelChange
}
