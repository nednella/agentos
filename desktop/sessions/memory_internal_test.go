package sessions

import "testing"

func TestHeldSumsAProcessAndEveryProcessUnderIt(t *testing.T) {
	sizes := map[int]int64{1: 10000, 100: 1000, 101: 2000, 102: 4000, 200: 8000}
	procs := readProcesses([]byte(`    1     0
  100     1
  101   100
  102   101
  200     1
  bad line
`), func(pid int) int64 { return sizes[pid] })
	for pid, want := range map[int]int64{100: 7000, 101: 6000, 200: 8000, 999: 0, 0: 0} {
		if got := procs.held(pid); got != want {
			t.Errorf("held(%d) = %d, want %d", pid, got, want)
		}
	}
}

func TestRoundUpKeepsAnyUseVisible(t *testing.T) {
	for n, want := range map[int64]int64{0: 0, 1: memoryStep, memoryStep: memoryStep, memoryStep + 1: 2 * memoryStep} {
		if got := roundUp(n); got != want {
			t.Errorf("roundUp(%d) = %d, want %d", n, got, want)
		}
	}
}
