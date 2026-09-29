package domain

// WholeMinutes floors seconds into whole minutes. It serves display
// ("stop_minutes", "route_total_minutes" — RN02/RN03) and the
// min_stop_minutes threshold comparison (RF10); it is never used to
// aggregate. Undefined for negative inputs — validation rejects them
// upstream.
func WholeMinutes(seconds int) int {
	return seconds / 60
}

// RouteTotalSeconds (RN03): the sum of counted stops' seconds (order > 1),
// honoring min_stop_minutes — a counted stop whose whole minutes fall
// below the threshold still records its timestamps but contributes 0
// (RF10; the default threshold 0 keeps the sum pure). Seconds are summed
// exactly; the caller floors once for display via WholeMinutes — never
// sum per-stop floored minutes.
func RouteTotalSeconds(r Route, minStopMinutes int) int {
	total := 0
	for _, s := range r.Stops {
		if s.Order == 1 {
			continue // RN01
		}
		secs := StopSeconds(s)
		if minStopMinutes > 0 && WholeMinutes(secs) < minStopMinutes {
			continue // Below threshold: contributes 0.
		}
		total += secs
	}
	return total
}
