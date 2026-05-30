package goncho

import "github.com/TrebuchetDynamics/goncho/service/internal/sliceutil"

func recallWarningListHasCode(warnings []RecallWarning, code string) bool {
	return sliceutil.ContainsFunc(warnings, func(warning RecallWarning) bool {
		return warning.Code == code
	})
}

func recallReplayEventsHaveWarning(events []RecallReplayEvent, code string) bool {
	return sliceutil.ContainsFunc(events, func(event RecallReplayEvent) bool {
		return event.WarningCode == code
	})
}

func recallTraceHasWarning(trace RecallTrace, code string) bool {
	return recallWarningListHasCode(trace.Warnings, code)
}

func recallReplayHasWarning(replay RecallReplay, code string) bool {
	return recallReplayEventsHaveWarning(replay.Events, code)
}
