package bench

import "astroswarm/worker/internal/sim"

func evaderSpeedMult(defenders []*sim.Ship, fallbackSpeed float64) float64 {
	speed := fallbackSpeed
	if len(defenders) > 0 {
		speed = defenders[0].MaxSpeed
	}
	return speed / sim.DefaultSpeed
}

func matchEvaderSpeeds(inFlight []*assaultEvader, defenders []*sim.Ship, fallbackSpeed float64) {
	speedMult := evaderSpeedMult(defenders, fallbackSpeed)
	for _, entry := range inFlight {
		if entry != nil {
			entry.ship.SpeedMult = speedMult
		}
	}
}
