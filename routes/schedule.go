package routes

import (
	"github.com/AlexGalhardo/vukem"
	"github.com/AlexGalhardo/vukem/schedule"
)

// Schedule declares the scheduled tasks. "vukem serve" runs them in process;
// with SCHEDULE_IN_SERVE=false run "vukem schedule:work" or a cron entry
// calling "vukem schedule:run" every minute instead. List them with
// "vukem schedule:list". Tasks must honor ctx: on shutdown it is cancelled
// after SHUTDOWN_TIMEOUT.
func Schedule(s *schedule.Schedule, _ *vukem.App) {
	// 00:30 UTC (SCHEDULE_TIMEZONE defaults to UTC) is after the US close in every season (21:00 UTC at
	// the latest) and after the UTC crypto day has closed, so both markets have a finished daily bar.
	s.Command("reco:daily").DailyAt("00:30").WithoutOverlapping().OnOneServer()
}
