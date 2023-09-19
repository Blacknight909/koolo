package run

import (
	"slices"
	"strings"
	"time"

	"github.com/hectorgimenez/d2go/pkg/data"
	"github.com/hectorgimenez/d2go/pkg/data/area"
	"github.com/hectorgimenez/d2go/pkg/data/item"
	"github.com/hectorgimenez/d2go/pkg/data/npc"
	"github.com/hectorgimenez/d2go/pkg/data/object"
	"github.com/hectorgimenez/koolo/internal/action"
	"github.com/hectorgimenez/koolo/internal/action/step"
)

type Cows struct {
	baseRun
}

func (a Cows) Name() string {
	return "Cows"
}

func (a Cows) BuildActions() (actions []action.Action) {
	return []action.Action{
		a.getWirtsLeg(),
		a.preparePortal(),
		a.builder.InteractObject(object.PermanentTownPortal, func(d data.Data) bool {
			return d.PlayerUnit.Area == area.MooMooFarm
		}),
		a.char.Buff(),
		a.builder.ClearArea(true, data.MonsterAnyFilter()),
	}
}

func (a Cows) getWirtsLeg() action.Action {
	return action.NewChain(func(d data.Data) []action.Action {
		if _, found := d.Items.Find("WirtsLeg", item.LocationStash, item.LocationInventory); found {
			a.logger.Info("WirtsLeg found, skip finding it")
			return nil
		}

		return []action.Action{
			a.builder.WayPoint(area.StonyField), // Moving to starting point (Stony Field)
			action.NewChain(func(d data.Data) []action.Action {
				for _, o := range d.Objects {
					if o.Name == object.CairnStoneAlpha {
						return []action.Action{a.builder.MoveToCoords(o.Position)}
					}
				}

				return nil
			}),
			a.builder.ClearAreaAroundPlayer(10),
			a.builder.ItemPickup(false, 15),
			a.builder.InteractObject(object.PermanentTownPortal, func(d data.Data) bool {
				return d.PlayerUnit.Area == area.Tristram
			}, step.Wait(time.Second)),
			a.builder.InteractObject(object.WirtCorpse, func(d data.Data) bool {
				_, found := d.Items.Find("WirtsLeg")

				return found
			}),
			a.builder.ItemPickup(false, 30),
			a.builder.ReturnTown(),
		}
	})
}

func (a Cows) preparePortal() action.Action {
	return action.NewChain(func(d data.Data) []action.Action {
		currentWPTomes := make([]data.UnitID, 0)
		leg, found := d.Items.Find("WirtsLeg")
		if !found {
			a.logger.Error("WirtsLeg could not be found, portal cannot be opened")
			return nil
		}

		// Backup current WP tomes in inventory, before getting new one at Akara
		for _, itm := range d.Items.ByLocation(item.LocationInventory) {
			if strings.EqualFold(string(itm.Name), item.TomeOfTownPortal) {
				currentWPTomes = append(currentWPTomes, itm.UnitID)
			}
		}

		actions := []action.Action{
			a.builder.BuyAtVendor(npc.Akara, action.VendorItemRequest{
				Item:     item.TomeOfTownPortal,
				Quantity: 1,
				Tab:      4,
			}),
			action.NewChain(func(d data.Data) []action.Action {
				// Ensure we are using the new WP tome and not the one that we are using for TPs
				var newWPTome data.Item
				for _, itm := range d.Items.ByLocation(item.LocationInventory) {
					if strings.EqualFold(string(itm.Name), item.TomeOfTownPortal) && !slices.Contains(currentWPTomes, itm.UnitID) {
						newWPTome = itm
					}
				}

				if newWPTome.UnitID == 0 {
					a.logger.Error("TomeOfTownPortal could not be found, portal cannot be opened")
					return nil
				}

				return []action.Action{
					a.builder.CubeAddItems(leg, newWPTome),
					a.builder.CubeTransmute(),
				}
			}),
		}

		return actions
	})
}