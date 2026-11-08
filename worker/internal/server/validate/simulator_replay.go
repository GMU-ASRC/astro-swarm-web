package validate

import (
	"fmt"

	"astroswarm/worker/internal/server/values"
)

func (submission *SimulatorSubmission) cleanReplay(replay map[string]any) error {
	interval, err := simulatorNumber(objectValue(replay, "interval", 0.1), "replay interval")
	if err != nil {
		return err
	}
	intervalValue := numberFloat(interval)
	if intervalValue < minRecordInterval || intervalValue > maxRecordInterval {
		return fmt.Errorf("replay interval must be between %s and %s seconds",
			values.FloatRepr(minRecordInterval), values.FloatRepr(maxRecordInterval))
	}
	robotsMessage := "replay.robots must be a list of [id, species id] pairs"
	robots, isList := objectValue(replay, "robots", []any{}).([]any)
	if !isList {
		return fmt.Errorf("%s", robotsMessage)
	}
	speciesIDs := submission.speciesIDs()
	robotSpecies := [][]any{}
	for _, item := range robots {
		pair, isList := item.([]any)
		if !isList || len(pair) != 2 {
			return fmt.Errorf("%s", robotsMessage)
		}
		robotID, err := integerNumber(pair[0], "robot id")
		if err != nil {
			return err
		}
		speciesID := values.ToText(pair[1])
		if !speciesIDs[speciesID] {
			speciesID = ""
		}
		robotSpecies = append(robotSpecies, []any{robotID, speciesID})
	}
	frames, isList := objectValue(replay, "frames", []any{}).([]any)
	if !isList || len(frames) < 1 || len(frames) > maxSimulatorFrames {
		return fmt.Errorf("replay.frames must hold between 1 and %d frames", maxSimulatorFrames)
	}
	cleanedFrames, peakRobots, err := cleanFrames(frames)
	if err != nil {
		return err
	}
	submission.RecordInterval = intervalValue
	submission.FrameCount = len(cleanedFrames)
	submission.PeakRobotCount = peakRobots
	submission.Replay = SimulatorReplay{Interval: interval, Robots: robotSpecies, Frames: cleanedFrames}
	return nil
}

func cleanFrames(frames []any) ([][]any, int, error) {
	samples, peakRobots := 0, 0
	cleanedFrames := [][]any{}
	labels := []string{"x", "y", "rotation"}
	for _, item := range frames {
		frame, isList := item.([]any)
		if !isList || len(frame)%valuesPerRobot != 0 {
			return nil, 0, fmt.Errorf("each replay frame must be a flat list of [id, x, y, rotation] groups")
		}
		robotsInFrame := len(frame) / valuesPerRobot
		samples += robotsInFrame
		if samples > maxRobotSamples {
			return nil, 0, fmt.Errorf("replay is too large: at most %d robot samples across all frames", maxRobotSamples)
		}
		peakRobots = max(peakRobots, robotsInFrame)
		cleanedFrame := make([]any, 0, len(frame))
		for offset := 0; offset < len(frame); offset += valuesPerRobot {
			robotID, err := integerNumber(frame[offset], "robot id")
			if err != nil {
				return nil, 0, err
			}
			cleanedFrame = append(cleanedFrame, robotID)
			for index, label := range labels {
				number, err := roundedNumber(frame[offset+1+index], label, 1)
				if err != nil {
					return nil, 0, err
				}
				cleanedFrame = append(cleanedFrame, number)
			}
		}
		cleanedFrames = append(cleanedFrames, cleanedFrame)
	}
	return cleanedFrames, peakRobots, nil
}
