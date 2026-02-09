package model

type WorkStatsTotals struct {
	Day   int64
	Week  int64
	Month int64
	All   int64
}

type WorkStatsActor struct {
	ActorTGID int64
	ActorRole string
	Username  string
	Day       int64
	Week      int64
	Month     int64
	All       int64
}

type WorkStatsReport struct {
	Totals WorkStatsTotals
	Actors []WorkStatsActor
}
