package runner

import "testing"

func TestSupervisorIsRunningUsesActiveProcessRegistry(t *testing.T) {
	supervisor := &Supervisor{
		processes: map[string]*ActiveProcess{
			"running-server": {},
		},
	}

	if !supervisor.IsRunning("running-server") {
		t.Fatal("expected active server to be reported as running")
	}
	if supervisor.IsRunning("stopped-server") {
		t.Fatal("expected unknown server to be reported as stopped")
	}
}

func TestNilSupervisorIsNotRunning(t *testing.T) {
	var supervisor *Supervisor

	if supervisor.IsRunning("server") {
		t.Fatal("expected nil supervisor to be reported as stopped")
	}
}
