package action

import (
	"fmt"
	"slices"
	"time"

	"github.com/hectorgimenez/koolo/internal/action/step"
	"github.com/hectorgimenez/koolo/internal/context"
	"github.com/hectorgimenez/koolo/internal/game"
	"github.com/hectorgimenez/koolo/internal/town"
	"github.com/hectorgimenez/koolo/internal/ui"
	"github.com/hectorgimenez/koolo/internal/utils"
	"github.com/lxn/win"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/area"
	"github.com/hectorgimenez/d2go/pkg/data/difficulty"
	"github.com/hectorgimenez/d2go/pkg/data/npc"
	"github.com/hectorgimenez/d2go/pkg/data/skill"
	"github.com/hectorgimenez/d2go/pkg/data/stat"
)

var uiStatButtonPosition = map[stat.ID]data.Position{
	stat.Strength:  {X: 240, Y: 210},
	stat.Dexterity: {X: 240, Y: 290},
	stat.Vitality:  {X: 240, Y: 380},
	stat.Energy:    {X: 240, Y: 430},
}

var uiSkillPagePosition = [3]data.Position{
	{X: 1100, Y: 140},
	{X: 1010, Y: 140},
	{X: 910, Y: 140},
}

var uiSkillRowPosition = [6]int{190, 250, 310, 365, 430, 490}
var uiSkillColumnPosition = [3]int{920, 1010, 1095}

var uiStatButtonPositionLegacy = map[stat.ID]data.Position{
	stat.Strength:  {X: 430, Y: 180},
	stat.Dexterity: {X: 430, Y: 250},
	stat.Vitality:  {X: 430, Y: 360},
	stat.Energy:    {X: 430, Y: 435},
}

var uiSkillPagePositionLegacy = [3]data.Position{
	{X: 970, Y: 510},
	{X: 970, Y: 390},
	{X: 970, Y: 260},
}

var uiSkillRowPositionLegacy = [6]int{110, 195, 275, 355, 440, 520}
var uiSkillColumnPositionLegacy = [3]int{690, 770, 855}

// EnsureStatPoints allocates stat points based on the target distribution
func EnsureStatPoints() error {
	ctx := context.Get()
	ctx.SetLastAction("EnsureStatPoints")

	char, isLevelingChar := ctx.Char.(context.LevelingCharacter)
	if !isLevelingChar {
		return nil
	}

	// Retrieve unused stat points
	_, unusedStatPoints := ctx.Data.PlayerUnit.FindStat(stat.StatPoints, 0)
	if !unusedStatPoints {
		// No stat points to allocate
		return nil
	}

	// Iterate over the target stat points
	for st, targetPoints := range char.StatPoints() {
		currentPoints, found := ctx.Data.PlayerUnit.FindStat(st, 0)
		if !found || currentPoints.Value >= targetPoints {
			continue
		}

		// Open the character screen if not already open
		if !ctx.Data.OpenMenus.Character {
			ctx.HID.PressKeyBinding(ctx.Data.KeyBindings.CharacterScreen)
			utils.Sleep(500) // Wait for the character screen to open
		}

		// Determine the position of the stat button
		var statBtnPosition data.Position
		if ctx.Data.LegacyGraphics {
			statBtnPosition = uiStatButtonPositionLegacy[st]
		} else {
			statBtnPosition = uiStatButtonPosition[st]
		}

		// Click on the stat button to allocate a point
		err := step.StepChain([]step.Step{
			step.SyncStep(func(d game.Data) error {
				utils.Sleep(100)
				ctx.HID.Click(game.LeftButton, statBtnPosition.X, statBtnPosition.Y)
				utils.Sleep(500) // Wait for the point to be allocated
				return nil
			}),
		}, step.RepeatUntilNoSteps())
		if err != nil {
			return fmt.Errorf("failed to allocate stat points for %v: %w", st, err)
		}
	}

	// Close the character screen if open
	if ctx.Data.OpenMenus.Character {
		err := step.CloseAllMenus()
		if err != nil {
			return fmt.Errorf("failed to close character screen: %w", err)
		}
	}

	return nil
}

// EnsureSkillPoints allocates skill points based on the Blizzard/Ice build
func EnsureSkillPoints() error {
	ctx := context.Get()
	ctx.SetLastAction("EnsureSkillPoints")

	char, isLevelingChar := ctx.Char.(context.LevelingCharacter)
	if !isLevelingChar {
		return nil
	}

	// Retrieve available skill points
	availablePoints, unusedSkillPoints := ctx.Data.PlayerUnit.FindStat(stat.SkillPoints, 0)
	if !unusedSkillPoints || availablePoints <= 0 {
		// No skill points to allocate
		return nil
	}

	assignedPoints := make(map[skill.ID]int)
	for _, sk := range char.SkillPoints() {
		currentPoints, found := assignedPoints[sk]
		if !found {
			currentPoints = 0
		}
		assignedPoints[sk] = currentPoints + 1

		characterPoints, found := ctx.Data.PlayerUnit.Skills[sk]
		if !found || int(characterPoints.Level) < assignedPoints[sk] {
			// Open the skill tree if not already open
			if !ctx.Data.OpenMenus.SkillTree {
				ctx.HID.PressKeyBinding(ctx.Data.KeyBindings.SkillTree)
				utils.Sleep(500) // Wait for the skill tree to open
			}

			// Navigate to the correct skill page
			skillDesc, skFound := skill.Desc[sk]
			if !skFound {
				ctx.Logger.Error("Skill description not found for skill", "skill", sk)
				continue
			}

			// Click on the skill page
			var skillPagePos data.Position
			if ctx.Data.LegacyGraphics {
				skillPagePos = uiSkillPagePositionLegacy[skillDesc.Page-1]
			} else {
				skillPagePos = uiSkillPagePosition[skillDesc.Page-1]
			}
			ctx.HID.Click(game.LeftButton, skillPagePos.X, skillPagePos.Y)
			utils.Sleep(200) // Wait for the page to load

			// Click on the specific skill
			var skillPos data.Position
			if ctx.Data.LegacyGraphics {
				skillPos = data.Position{
					X: uiSkillColumnPositionLegacy[skillDesc.Column-1],
					Y: uiSkillRowPositionLegacy[skillDesc.Row-1],
				}
			} else {
				skillPos = data.Position{
					X: uiSkillColumnPosition[skillDesc.Column-1],
					Y: uiSkillRowPosition[skillDesc.Row-1],
				}
			}
			ctx.HID.Click(game.LeftButton, skillPos.X, skillPos.Y)
			utils.Sleep(500) // Wait for the skill point to be allocated
		}
	}

	// Close the skill tree if open
	if ctx.Data.OpenMenus.SkillTree {
		err := step.CloseAllMenus()
		if err != nil {
			return fmt.Errorf("failed to close skill tree: %w", err)
		}
	}

	return nil
}

// UpdateQuestLog updates the quest log by closing it
func UpdateQuestLog() error {
	ctx := context.Get()
	ctx.SetLastAction("UpdateQuestLog")

	if _, isLevelingChar := ctx.Char.(context.LevelingCharacter); !isLevelingChar {
		return nil
	}

	// Open the quest log
	ctx.HID.PressKeyBinding(ctx.Data.KeyBindings.QuestLog)
	utils.Sleep(1000)

	// Close all menus including the quest log
	return step.CloseAllMenus()
}

// getAvailableSkillKB retrieves available key bindings for skills
func getAvailableSkillKB() []data.KeyBinding {
	availableSkillKB := make([]data.KeyBinding, 0)
	ctx := context.Get()
	ctx.SetLastAction("getAvailableSkillKB")

	for _, sb := range ctx.Data.KeyBindings.Skills {
		if sb.SkillID == -1 && ((sb.Key1[0] != 0 && sb.Key1[0] != 255) || (sb.Key2[0] != 0 && sb.Key2[0] != 255)) {
			availableSkillKB = append(availableSkillKB, sb.KeyBinding)
		}
	}

	return availableSkillKB
}

// EnsureSkillBindings ensures that all desired skills are bound to key bindings
func EnsureSkillBindings() error {
	ctx := context.Get()
	ctx.SetLastAction("EnsureSkillBindings")

	char, isLevelingChar := ctx.Char.(context.LevelingCharacter)
	if !isLevelingChar {
		return nil
	}

	mainSkill, skillsToBind := char.SkillsToBind()
	// Ensure Tome of Town Portal is always bound
	if !slices.Contains(skillsToBind, skill.TomeOfTownPortal) && ctx.Data.PlayerUnit.Skills[skill.TomeOfTownPortal].Level > 0 {
		skillsToBind = append(skillsToBind, skill.TomeOfTownPortal)
	}

	// Identify skills that are not yet bound
	notBoundSkills := make([]skill.ID, 0)
	for _, sk := range skillsToBind {
		if _, found := ctx.Data.KeyBindings.KeyBindingForSkill(sk); !found && ctx.Data.PlayerUnit.Skills[sk].Level > 0 {
			notBoundSkills = append(notBoundSkills, sk)
		}
	}

	if len(notBoundSkills) > 0 {
		// Open the secondary skill binding menu
		ctx.HID.Click(game.LeftButton, ui.SecondarySkillButtonX, ui.SecondarySkillButtonY)
		utils.Sleep(300)
		ctx.HID.MovePointer(10, 10) // Move cursor away to prevent accidental clicks
		utils.Sleep(300)

		availableKB := getAvailableSkillKB()

		for i, sk := range notBoundSkills {
			if i >= len(availableKB) {
				ctx.Logger.Warn("Not enough available key bindings to bind all skills")
				break
			}

			skillPosition, found := calculateSkillPositionInUI(false, sk)
			if !found {
				ctx.Logger.Warn("Skill position not found for skill", "skill", sk)
				continue
			}

			// Move pointer to the skill and press the available key binding
			ctx.HID.MovePointer(skillPosition.X, skillPosition.Y)
			utils.Sleep(100)
			ctx.HID.PressKeyBinding(availableKB[i])
			utils.Sleep(300)
		}
	}

	// Ensure the main skill is correctly bound
	if ctx.Data.PlayerUnit.LeftSkill != mainSkill {
		ctx.HID.Click(game.LeftButton, ui.MainSkillButtonX, ui.MainSkillButtonY)
		utils.Sleep(300)
		ctx.HID.MovePointer(10, 10) // Move cursor away
		utils.Sleep(300)

		skillPosition, found := calculateSkillPositionInUI(true, mainSkill)
		if found {
			ctx.HID.MovePointer(skillPosition.X, skillPosition.Y)
			utils.Sleep(100)
			ctx.HID.Click(game.LeftButton, skillPosition.X, skillPosition.Y)
			utils.Sleep(300)
		} else {
			ctx.Logger.Warn("Main skill position not found", "mainSkill", mainSkill)
		}
	}

	return nil
}

// calculateSkillPositionInUI calculates the UI position for a given skill
func calculateSkillPositionInUI(mainSkill bool, skillID skill.ID) (data.Position, bool) {
	d := context.Get().Data

	// List of scroll skills to exclude
	var scrolls = []skill.ID{
		skill.TomeOfTownPortal, skill.ScrollOfTownPortal, skill.TomeOfIdentify, skill.ScrollOfIdentify,
	}

	// Verify the skill exists
	if _, found := d.PlayerUnit.Skills[skillID]; !found {
		return data.Position{}, false
	}

	targetSkill := skill.Skills[skillID]
	skillDesc := targetSkill.Desc()

	// Skip skills that cannot be bound
	if skillDesc.ListRow < 0 {
		return data.Position{}, false
	}

	// Skip skills that cannot be bound to the current mouse button
	if (mainSkill && !targetSkill.LeftSkill) || (!mainSkill && !targetSkill.RightSkill) {
		return data.Position{}, false
	}

	// Calculate the column and row based on skill description
	column := skillDesc.Column - 1
	row := skillDesc.Row - 1

	// Determine the skill page position
	var skillPagePos data.Position
	if d.LegacyGraphics {
		skillPagePos = uiSkillPagePositionLegacy[skillDesc.Page-1]
	} else {
		skillPagePos = uiSkillPagePosition[skillDesc.Page-1]
	}

	// Determine the skill position within the page
	var skillPos data.Position
	if d.LegacyGraphics {
		skillPos = data.Position{
			X: uiSkillColumnPositionLegacy[column],
			Y: uiSkillRowPositionLegacy[row],
		}
	} else {
		skillPos = data.Position{
			X: uiSkillColumnPosition[column],
			Y: uiSkillRowPosition[row],
		}
	}

	return skillPos, true
}

// HireMerc hires a mercenary based on the configuration
func HireMerc() error {
	ctx := context.Get()
	ctx.SetLastAction("HireMerc")

	_, isLevelingChar := ctx.Char.(context.LevelingCharacter)
	if isLevelingChar && ctx.CharacterCfg.Character.UseMerc {
		// Hire the mercenary if:
		// - Difficulty is Normal
		// - No mercenary is currently active
		// - Player has more than 30,000 gold
		// - Player is in Lut Gholein
		if ctx.CharacterCfg.Game.Difficulty == difficulty.Normal &&
			ctx.Data.MercHPPercent() <= 0 &&
			ctx.Data.PlayerUnit.TotalPlayerGold() > 30000 &&
			ctx.Data.PlayerUnit.Area == area.LutGholein {
			ctx.Logger.Info("Hiring mercenary...")

			// Interact with the mercenary contractor NPC
			err := InteractNPC(town.GetTownByArea(ctx.Data.PlayerUnit.Area).MercContractorNPC())
			if err != nil {
				return fmt.Errorf("failed to interact with mercenary contractor: %w", err)
			}

			// Navigate the mercenary hiring menu
			ctx.HID.KeySequence(win.VK_HOME, win.VK_DOWN, win.VK_RETURN)
			utils.Sleep(2000)

			// Click on the first mercenary in the list (prefer Holy Freeze)
			// TODO: Implement selection logic for Holy Freeze mercenary
			ctx.HID.Click(game.LeftButton, ui.FirstMercFromContractorListX, ui.FirstMercFromContractorListY)
			utils.Sleep(500)
			ctx.HID.Click(game.LeftButton, ui.FirstMercFromContractorListX, ui.FirstMercFromContractorListY)
			utils.Sleep(500)
		}
	}

	return nil
}

// ResetStats resets the Sorceress skills if conditions are met
func ResetStats() error {
	ctx := context.Get()
	ctx.SetLastAction("ResetStats")

	ch, isLevelingChar := ctx.Char.(context.LevelingCharacter)
	if isLevelingChar && ch.ShouldResetSkills() {
		currentArea := ctx.Data.PlayerUnit.Area
		// Navigate to Rogue Encampment if not already there
		if ctx.Data.PlayerUnit.Area != area.RogueEncampment {
			err := WayPoint(area.RogueEncampment)
			if err != nil {
				return fmt.Errorf("failed to waypoint to Rogue Encampment: %w", err)
			}
		}

		// Interact with Akara to reset skills
		err := InteractNPC(npc.Akara)
		if err != nil {
			return fmt.Errorf("failed to interact with Akara: %w", err)
		}

		// Navigate the skill reset menu
		ctx.HID.KeySequence(win.VK_HOME, win.VK_DOWN, win.VK_DOWN, win.VK_RETURN)
		utils.Sleep(1000)
		ctx.HID.KeySequence(win.VK_HOME, win.VK_RETURN)
		utils.Sleep(500)

		// Return to the original area if it was changed
		if currentArea != area.RogueEncampment {
			err := WayPoint(currentArea)
			if err != nil {
				return fmt.Errorf("failed to waypoint back to original area: %w", err)
			}
		}
	}

	return nil
}

// InteractNPC interacts with a specified NPC
func InteractNPC(npcID npc.ID) error {
	ctx := context.Get()
	interactable, found := ctx.Data.NPCs.FindByID(npcID)
	if !found {
		return fmt.Errorf("NPC with ID %v not found", npcID)
	}

	// Move to the NPC's position
	err := step.MoveTo(interactable.Position)
	if err != nil {
		return fmt.Errorf("failed to move to NPC position: %w", err)
	}

	// Click on the NPC to interact
	ctx.HID.Click(game.LeftButton, interactable.Position.X, interactable.Position.Y)
	utils.Sleep(500) // Wait for the interaction menu to open

	return nil
}

// WayPoint navigates to a specified area using waypoints
func WayPoint(targetArea area.ID) error {
	ctx := context.Get()
	currentArea := ctx.Data.PlayerUnit.Area

	// Define waypoint coordinates for relevant areas
	waypoints := map[area.ID]data.Position{
		area.RogueEncampment: {X: 900, Y: 900}, // Example coordinates
		area.LutGholein:       {X: 1200, Y: 1200},
		// Add other necessary waypoints here
	}

	waypointPos, exists := waypoints[targetArea]
	if !exists {
		return fmt.Errorf("waypoint for area %v not defined", targetArea)
	}

	// Move to the waypoint
	err := step.MoveTo(waypointPos)
	if err != nil {
		return fmt.Errorf("failed to move to waypoint: %w", err)
	}

	utils.Sleep(1000) // Wait for the area to load

	return nil
}
