package model

import "time"

// MedianTimeSource provides a mechanism to add several time samples which are
// used to determine a median time which is then used as an offset to the local
// clock.
type MedianTimeSource interface {
	// AdjustedTime returns the current time adjusted by the median time
	// offset as calculated from the time samples added by AddTimeSample.
	AdjustedTime() time.Time

	// AddTimeSample adds a time sample that is used when determining the
	// median time of the added samples.
	AddTimeSample(id string, timeVal time.Time)

	// Offset returns the number of seconds to adjust the local clock based
	// upon the median of the time samples added by AddTimeData.
	Offset() time.Duration
}
