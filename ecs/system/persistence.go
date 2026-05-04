package system

import (
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/jakecoffman/cp"
	"github.com/milk9111/sidescroller/ecs"
	"github.com/milk9111/sidescroller/ecs/component"
	"github.com/milk9111/sidescroller/ecs/entity"
	"github.com/milk9111/sidescroller/internal/savegame"
	"github.com/milk9111/sidescroller/levels"
)

const playerAttackHitEmitterName = "player_attack_hit"

type PersistenceMode int

const (
	PersistenceOnLevelChange PersistenceMode = iota
	PersistenceOnReload
)

type PersistenceSystem struct {
	levelName        string
	initialLevelName string
	allAbilities     bool
	initialAbilities *component.Abilities
	initialFadeIn    bool
	physicsReset     func()
	saveStore        *savegame.Store
	loadedSave       *savegame.File
	initialized      bool
	loadSequence     uint64
}

func NewPersistenceSystem(initialLevelName string, allAbilities bool, initialAbilities *component.Abilities, initialFadeIn bool, physicsReset func(), saveStore *savegame.Store, loadedSave *savegame.File) *PersistenceSystem {
	levelName := initialLevelName
	if loadedSave != nil && loadedSave.Level != "" {
		levelName = loadedSave.Level
	}

	return &PersistenceSystem{
		levelName:        levelName,
		initialLevelName: levelName,
		allAbilities:     allAbilities,
		initialAbilities: initialAbilities,
		initialFadeIn:    initialFadeIn,
		physicsReset:     physicsReset,
		saveStore:        saveStore,
		loadedSave:       loadedSave,
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
		p.queueAsyncSave(w)
		if p.initialFadeIn {
			rtEnt := ecs.CreateEntity(w)
			_ = ecs.Add(w, rtEnt, component.TransitionRuntimeComponent.Kind(), &component.TransitionRuntime{
				Phase:   component.TransitionFadeIn,
				Alpha:   1,
				Timer:   transitionFadeFrames,
				Req:     component.LevelChangeRequest{},
				ReqSent: true,
			})
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
		p.queueAsyncSave(w)
		return
	}

	if _, ok := ecs.First(w, component.ReloadRequestComponent.Kind()); ok {
		ecs.ForEach(w, component.ReloadRequestComponent.Kind(), func(e ecs.Entity, _ *component.ReloadRequest) {
			ecs.DestroyEntity(w, e)
		})
		if err := p.reloadWorld(w, PersistenceOnReload); err != nil {
			panic("persistence system: reload failed: " + err.Error())
		}
		p.queueAsyncSave(w)
		return
	}

	if req, ok := p.firstCheckpointReloadRequest(w); ok {
		if p.checkpointReloadWaitingForFade(w) {
			return
		}
		ecs.ForEach(w, component.CheckpointReloadRequestComponent.Kind(), func(e ecs.Entity, _ *component.CheckpointReloadRequest) {
			ecs.DestroyEntity(w, e)
		})
		if err := p.reloadCheckpoint(w, req); err != nil {
			panic("persistence system: checkpoint reload failed: " + err.Error())
		}
		return
	}

	if _, ok := p.firstEnemyRespawnRequest(w); ok {
		ecs.ForEach(w, component.EnemyRespawnRequestComponent.Kind(), func(e ecs.Entity, _ *component.EnemyRespawnRequest) {
			ecs.DestroyEntity(w, e)
		})
		if err := p.respawnCurrentLevelEnemies(w); err != nil {
			panic("persistence system: enemy respawn reload failed: " + err.Error())
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

		p.spawnPlayerAtLinkedTransition(w, req)
		p.applyTransitionPop(w, req)
		p.queueAsyncSave(w)

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

func hasParticleEmitterNamed(w *ecs.World, name string) bool {
	if w == nil || name == "" {
		return false
	}
	found := false
	ecs.ForEach(w, component.ParticleEmitterComponent.Kind(), func(_ ecs.Entity, emitter *component.ParticleEmitter) {
		if found || emitter == nil {
			return
		}
		if emitter.Name == name {
			found = true
		}
	})
	return found
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
				if e != preferredEntity {
					p.mergePersistentLevelScopedComponents(w, preferredEntity, e)
				}
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

func (p *PersistenceSystem) mergePersistentLevelScopedComponents(w *ecs.World, dst, src ecs.Entity) {
	if w == nil || !ecs.IsAlive(w, dst) || !ecs.IsAlive(w, src) {
		return
	}

	if layer, ok := ecs.Get(w, src, component.EntityLayerComponent.Kind()); ok && layer != nil {
		copied := *layer
		_ = ecs.Add(w, dst, component.EntityLayerComponent.Kind(), &copied)
	}

	if id, ok := ecs.Get(w, src, component.GameEntityIDComponent.Kind()); ok && id != nil {
		copied := *id
		_ = ecs.Add(w, dst, component.GameEntityIDComponent.Kind(), &copied)
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

func (p *PersistenceSystem) firstCheckpointReloadRequest(w *ecs.World) (component.CheckpointReloadRequest, bool) {
	ent, ok := ecs.First(w, component.CheckpointReloadRequestComponent.Kind())
	if !ok {
		return component.CheckpointReloadRequest{}, false
	}
	req, ok := ecs.Get(w, ent, component.CheckpointReloadRequestComponent.Kind())
	if !ok || req == nil {
		return component.CheckpointReloadRequest{}, false
	}
	return *req, true
}

func (p *PersistenceSystem) firstEnemyRespawnRequest(w *ecs.World) (component.EnemyRespawnRequest, bool) {
	ent, ok := ecs.First(w, component.EnemyRespawnRequestComponent.Kind())
	if !ok {
		return component.EnemyRespawnRequest{}, false
	}
	req, ok := ecs.Get(w, ent, component.EnemyRespawnRequestComponent.Kind())
	if !ok || req == nil {
		return component.EnemyRespawnRequest{}, false
	}
	return *req, true
}

func (p *PersistenceSystem) checkpointReloadWaitingForFade(w *ecs.World) bool {
	if w == nil {
		return false
	}
	rtEnt, ok := ecs.First(w, component.TransitionRuntimeComponent.Kind())
	if !ok {
		return false
	}
	rt, ok := ecs.Get(w, rtEnt, component.TransitionRuntimeComponent.Kind())
	if !ok || rt == nil {
		return false
	}
	return rt.Phase == component.TransitionFadeOut && rt.Timer > 0
}

func (p *PersistenceSystem) reloadCheckpoint(w *ecs.World, req component.CheckpointReloadRequest) error {
	if p == nil || w == nil {
		return nil
	}

	snapshot, err := p.checkpointSnapshot(w, req.SaveBeforeReload)
	if err != nil {
		return err
	}
	if snapshot == nil {
		return nil
	}

	checkpoint := effectiveCheckpoint(snapshot)
	if checkpoint.Initialized && strings.TrimSpace(checkpoint.Level) != "" {
		p.levelName = checkpoint.Level
	} else if strings.TrimSpace(snapshot.Level) != "" {
		p.levelName = snapshot.Level
	}

	p.loadedSave = snapshot
	if err := p.reloadWorld(w, PersistenceOnReload); err != nil {
		return err
	}
	applyCheckpointRespawn(w, checkpoint)
	if err := p.respawnCurrentLevelEnemies(w); err != nil {
		return err
	}
	p.queueAsyncSave(w)
	return nil
}

func (p *PersistenceSystem) respawnCurrentLevelEnemies(w *ecs.World) error {
	if p == nil || w == nil {
		return nil
	}
	levelName := strings.TrimSpace(currentLevelName(w))
	if levelName == "" {
		return nil
	}

	runtimeEntity, ok := ecs.First(w, component.LevelRuntimeComponent.Kind())
	if !ok {
		return nil
	}
	runtimeComp, ok := ecs.Get(w, runtimeEntity, component.LevelRuntimeComponent.Kind())
	if !ok || runtimeComp == nil || runtimeComp.Level == nil {
		return nil
	}

	stateMap := ensurePlayerLevelEntityStateMap(w)
	if stateMap == nil {
		return nil
	}

	if _, err := entity.ResetCurrentLevelEnemies(w, levelName, runtimeComp.Level, stateMap); err != nil {
		return err
	}
	p.queueAsyncSave(w)
	return nil
}

func (p *PersistenceSystem) checkpointSnapshot(w *ecs.World, saveBeforeReload bool) (*savegame.File, error) {
	if saveBeforeReload {
		snapshot, err := savegame.CaptureWorld(w)
		if err != nil {
			return nil, err
		}
		if p.saveStore != nil {
			if err := p.saveStore.Save(snapshot); err != nil {
				return nil, err
			}
		}
		return snapshot, nil
	}

	if p.saveStore != nil {
		snapshot, err := p.saveStore.Load()
		if err == nil && snapshot != nil {
			return snapshot, nil
		}
	}

	return savegame.CaptureWorld(w)
}

func effectiveCheckpoint(snapshot *savegame.File) savegame.CheckpointState {
	if snapshot == nil {
		return savegame.CheckpointState{}
	}
	if snapshot.Player.Checkpoint.Initialized {
		return snapshot.Player.Checkpoint
	}
	return savegame.CheckpointState{
		Level:       snapshot.Level,
		X:           snapshot.Player.SafeRespawn.X,
		Y:           snapshot.Player.SafeRespawn.Y,
		FacingLeft:  snapshot.Player.FacingLeft,
		Health:      snapshot.Player.Health.Initial,
		HealUses:    0,
		Initialized: snapshot.Player.SafeRespawn.Initialized,
	}
}

func applyCheckpointRespawn(w *ecs.World, checkpoint savegame.CheckpointState) {
	if w == nil || !checkpoint.Initialized {
		return
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}

	if transform, ok := ecs.Get(w, player, component.TransformComponent.Kind()); ok && transform != nil {
		transform.X = checkpoint.X
		transform.Y = checkpoint.Y
		_ = ecs.Add(w, player, component.TransformComponent.Kind(), transform)
	}
	if sprite, ok := ecs.Get(w, player, component.SpriteComponent.Kind()); ok && sprite != nil {
		sprite.FacingLeft = checkpoint.FacingLeft
		_ = ecs.Add(w, player, component.SpriteComponent.Kind(), sprite)
	}
	if health, ok := ecs.Get(w, player, component.HealthComponent.Kind()); ok && health != nil {
		if checkpoint.Health > 0 {
			health.Current = checkpoint.Health
		}
		_ = ecs.Add(w, player, component.HealthComponent.Kind(), health)
	}
	if stateMachine, ok := ecs.Get(w, player, component.PlayerStateMachineComponent.Kind()); ok && stateMachine != nil {
		stateMachine.HealUses = checkpoint.HealUses
		stateMachine.DeathTimer = 0
		stateMachine.State = nil
		stateMachine.Pending = nil
		_ = ecs.Add(w, player, component.PlayerStateMachineComponent.Kind(), stateMachine)
	}
	_ = ecs.Add(w, player, component.SafeRespawnComponent.Kind(), &component.SafeRespawn{X: checkpoint.X, Y: checkpoint.Y, Initialized: true})
	_ = ecs.Add(w, player, component.PlayerCheckpointComponent.Kind(), &component.PlayerCheckpoint{
		Level:       checkpoint.Level,
		X:           checkpoint.X,
		Y:           checkpoint.Y,
		FacingLeft:  checkpoint.FacingLeft,
		Health:      checkpoint.Health,
		HealUses:    checkpoint.HealUses,
		Initialized: true,
	})
	if body, ok := ecs.Get(w, player, component.PhysicsBodyComponent.Kind()); ok && body != nil && body.Body != nil {
		if transform, ok := ecs.Get(w, player, component.TransformComponent.Kind()); ok && transform != nil {
			centerX := bodyCenterX(w, player, transform, body)
			centerY := bodyCenterY(transform, body)
			body.Body.SetPosition(cp.Vector{X: centerX, Y: centerY})
			body.Body.SetVelocityVector(cp.Vector{})
			body.Body.SetAngularVelocity(0)
			_ = ecs.Add(w, player, component.PhysicsBodyComponent.Kind(), body)
		}
	}
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
	if runtimeEnt, ok := ecs.First(w, component.LevelRuntimeComponent.Kind()); ok {
		if runtimeComp, ok := ecs.Get(w, runtimeEnt, component.LevelRuntimeComponent.Kind()); ok && runtimeComp != nil {
			runtimeComp.Name = name
			_ = ecs.Add(w, runtimeEnt, component.LevelRuntimeComponent.Kind(), runtimeComp)
		}
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

	if p.loadedSave != nil {
		if err := savegame.ApplyWorld(w, p.loadedSave); err != nil {
			return fmt.Errorf("apply loaded save: %w", err)
		}
		if !playerHasActiveTransitionCooldown(w) {
			p.armTransitionCooldownForCurrentOverlap(w)
		}
		p.loadedSave = nil
	}

	ensurePlayerLevelEntityStateMap(w)
	ensurePlayerLevelLayerStateMap(w)
	if err := applyPersistedLevelLayerStates(w); err != nil {
		return err
	}
	applyPersistedLevelEntityStates(w)

	if _, ok := ecs.First(w, component.AimTargetTagComponent.Kind()); !ok {
		if _, err = entity.BuildEntity(w, "aim_target.yaml"); err != nil {
			return err
		}
	}

	if _, ok := ecs.First(w, component.MusicPlayerComponent.Kind()); !ok {
		if _, err = entity.NewMusicPlayer(w); err != nil {
			return err
		}
	}

	if _, ok := ecs.First(w, component.DialoguePopupComponent.Kind()); !ok {
		if _, err = entity.NewDialoguePopup(w); err != nil {
			return err
		}
	}

	if _, ok := ecs.First(w, component.ShrinePopupComponent.Kind()); !ok {
		if _, err = entity.NewShrinePopup(w); err != nil {
			return err
		}
	}

	if _, ok := ecs.First(w, component.ItemPopupComponent.Kind()); !ok {
		if _, err = entity.NewItemPopup(w); err != nil {
			return err
		}
	}

	if _, ok := ecs.First(w, component.TransitionPopupComponent.Kind()); !ok {
		if _, err = entity.NewTransitionPopup(w); err != nil {
			return err
		}
	}

	if !hasParticleEmitterNamed(w, playerAttackHitEmitterName) {
		if _, err = entity.BuildEntity(w, "emitter_player_attack_hit.yaml"); err != nil {
			return err
		}
	}

	if _, ok := ecs.First(w, component.UIRootComponent.Kind()); !ok {
		if _, err = entity.NewUIRoot(w); err != nil {
			return err
		}
	}

	if _, ok := ecs.First(w, component.PlayerHealthBarComponent.Kind()); !ok {
		if _, err = entity.NewPlayerHealthBar(w); err != nil {
			return err
		}
	}

	if _, ok := ecs.First(w, component.DebugMessageComponent.Kind()); !ok {
		if _, err = entity.NewDebugMessage(w); err != nil {
			return err
		}
	}

	if _, ok := ecs.First(w, component.AbilitiesComponent.Kind()); !ok {
		abEnt := ecs.CreateEntity(w)
		_ = ecs.Add(w, abEnt, component.PersistentComponent.Kind(), &component.Persistent{
			ID:                "player_abilities",
			KeepOnLevelChange: true,
			KeepOnReload:      true,
		})
		// Use explicit initial abilities if provided, otherwise fall back to the allAbilities flag
		if p.initialAbilities != nil {
			a := *p.initialAbilities
			_ = ecs.Add(w, abEnt, component.AbilitiesComponent.Kind(), &a)
		} else {
			_ = ecs.Add(w, abEnt, component.AbilitiesComponent.Kind(), &component.Abilities{
				DoubleJump: p.allAbilities,
				WallGrab:   p.allAbilities,
				Anchor:     p.allAbilities,
				Heal:       p.allAbilities,
			})
		}
	}

	ensurePlayerGearCount(w)

	p.loadSequence++
	ent := ecs.CreateEntity(w)
	_ = ecs.Add(w, ent, component.LevelLoadedComponent.Kind(), &component.LevelLoaded{Sequence: p.loadSequence})
	return nil
}

func (p *PersistenceSystem) spawnPlayerAtLinkedTransition(w *ecs.World, req component.LevelChangeRequest) {
	if w == nil {
		return
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}

	var (
		spawnX               float64
		spawnY               float64
		spawnH               float64
		resolvedTransitionID string
		found                bool
		isLeft               bool
		isRight              bool
	)

	assignSpawn := func(tr *component.Transition, tf *component.Transform) {
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
		resolvedTransitionID = tr.ID
		found = true
		isLeft = tr.EnterDir == component.TransitionDirLeft
		isRight = tr.EnterDir == component.TransitionDirRight
	}

	ecs.ForEach2(w, component.TransitionComponent.Kind(), component.TransformComponent.Kind(), func(_ ecs.Entity, tr *component.Transition, tf *component.Transform) {
		if found || tr == nil || tf == nil || req.SpawnTransitionID == "" || tr.ID != req.SpawnTransitionID {
			return
		}
		assignSpawn(tr, tf)
	})

	if !found && req.SpawnTransitionID != "" && req.FromTransitionID != "" {
		ecs.ForEach2(w, component.TransitionComponent.Kind(), component.TransformComponent.Kind(), func(_ ecs.Entity, tr *component.Transition, tf *component.Transform) {
			if found || tr == nil || tf == nil || tr.LinkedID != req.FromTransitionID {
				return
			}
			assignSpawn(tr, tf)
		})
	}

	if !found {
		return
	}

	playerTf, ok := ecs.Get(w, player, component.TransformComponent.Kind())
	if !ok || playerTf == nil {
		playerTf = &component.Transform{ScaleX: 1, ScaleY: 1}
	}

	if playerBody, ok := ecs.Get(w, player, component.PhysicsBodyComponent.Kind()); ok && playerBody != nil && playerBody.Width > 0 && playerBody.Height > 0 {
		centerOffsetX := playerBody.OffsetX
		centerOffsetY := playerBody.OffsetY
		if playerBody.AlignTopLeft {
			centerOffsetX += playerBody.Width / 2
			centerOffsetY += playerBody.Height / 2
		}
		if !isLeft && !isRight {
			playerTf.X = spawnX - centerOffsetX
			playerTf.Y = spawnY - centerOffsetY
		} else {
			playerTf.X = spawnX - centerOffsetX
			playerTf.Y = spawnY + spawnH/2 - (centerOffsetY + playerBody.Height/2)
		}
	} else {
		playerTf.X = spawnX
		playerTf.Y = spawnY
	}
	_ = ecs.Add(w, player, component.TransformComponent.Kind(), playerTf)

	if playerSprite, ok := ecs.Get(w, player, component.SpriteComponent.Kind()); ok && playerSprite != nil {
		playerSprite.FacingLeft = isLeft
	}

	p.armTransitionCooldownForCurrentOverlap(w)
	if cooldown, ok := ecs.Get(w, player, component.TransitionCooldownComponent.Kind()); ok && cooldown != nil {
		if cooldown.TransitionID == "" {
			cooldown.TransitionID = resolvedTransitionID
		}
		if len(cooldown.TransitionIDs) == 0 && resolvedTransitionID != "" {
			cooldown.TransitionIDs = []string{resolvedTransitionID}
		}
		cooldown.Active = true
		_ = ecs.Add(w, player, component.TransitionCooldownComponent.Kind(), cooldown)
		return
	}

	_ = ecs.Add(w, player, component.TransitionCooldownComponent.Kind(), &component.TransitionCooldown{
		Active:        true,
		TransitionID:  resolvedTransitionID,
		TransitionIDs: []string{resolvedTransitionID},
	})
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
		VY:          -jp * 2.0,
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

func (p *PersistenceSystem) queueAsyncSave(w *ecs.World) {
	if p == nil || p.saveStore == nil || w == nil {
		return
	}

	snapshot, err := savegame.CaptureWorld(w)
	if err != nil {
		log.Printf("save game: capture snapshot: %v", err)
		return
	}

	p.saveStore.SaveAsync(snapshot)
}

func (p *PersistenceSystem) armTransitionCooldownForCurrentOverlap(w *ecs.World) {
	if w == nil {
		return
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return
	}

	playerBounds, ok := playerAABB(w, player)
	if !ok {
		return
	}

	transitionIDs := make([]string, 0, 2)
	ecs.ForEach2(w, component.TransitionComponent.Kind(), component.TransformComponent.Kind(), func(e ecs.Entity, tr *component.Transition, _ *component.Transform) {
		if tr == nil || tr.TargetLevel == "" || tr.LinkedID == "" || component.NormalizeTransitionType(tr.Type) == component.TransitionTypeInside {
			return
		}
		if aabbIntersects(playerBounds, transitionAABB(w, e, tr)) {
			transitionIDs = append(transitionIDs, tr.ID)
		}
	})

	if len(transitionIDs) == 0 {
		return
	}

	_ = ecs.Add(w, player, component.TransitionCooldownComponent.Kind(), &component.TransitionCooldown{
		Active:        true,
		TransitionID:  transitionIDs[0],
		TransitionIDs: transitionIDs,
	})
}

func playerHasActiveTransitionCooldown(w *ecs.World) bool {
	if w == nil {
		return false
	}

	player, ok := ecs.First(w, component.PlayerTagComponent.Kind())
	if !ok {
		return false
	}

	cooldown, ok := ecs.Get(w, player, component.TransitionCooldownComponent.Kind())
	return ok && cooldown != nil && cooldown.Active
}
