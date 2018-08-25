package server

/*
func init() {
	SetupCheckServiceLogger(&LogSettings{Level: "warn", Stdout: true})
	ensureTestDB()
}

func mustHaveCommand(t *testing.T, bin string) *exec.Cmd {
	binPath, err := exec.LookPath(bin)
	if err != nil {
		t.Skip("Command not found for test:", err)
	}
	return exec.Command(binPath)
}

func expectedCheckResult(team models.Team, check models.Check, exitCode int, timestamp time.Time) models.Result {
	res := models.Result{
		Type:       "Service",
		Teamname:   team.Name,
		Teamnumber: team.Number,
		Group:      check.Name,
		Details:    fmt.Sprintf("Status: %d", exitCode),
		Timestamp:  timestamp,
	}
	if exitCode < len(check.Points) {
		res.Points = check.Points[exitCode]
	}
	return res
}

func Test_runCmd(t *testing.T) {
	tests := map[string]struct {
		commandName string
		args        string
		exitCode    int
	}{
		// echo should be available everywhere
		"exit 0": {"echo", "", 0},
		// whoami will exit 1 with unknown arg
		"exit 1": {"whoami", "--nil", 1},
		// test the argument replacement; `arp` is available everywhere, where `ping` may be restricted
		"with args": {"arp", "IP", 0},
		// If the script goes missing, runCmd should send a no-score result on the chan
		"missing cmd": {"gone-fishing", "", 127},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			check := models.Check{
				Name:   name,
				Script: exec.Command(tt.commandName),
				Args:   tt.args,
				Points: []int{1, 0},
			}

			if tt.exitCode != 127 {
				mustHaveCommand(t, tt.commandName)
			}

			team, timestamp, timeout := TestTeams[0], time.Time{}, time.Second
			status := make(chan models.Result)

			go runCmd(team, check, timestamp, timeout, status)

			select {
			case res := <-status:
				expect := expectedCheckResult(team, check, tt.exitCode, timestamp)
				assert.Equal(t, expect, res)
			case <-time.After(timeout):
				t.Log("timed out!")
				t.FailNow()
			}
		})
	}
}
*/
