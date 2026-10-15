package sessions

import "testing"

func TestHeldSumsAProcessAndEveryProcessUnderIt(t *testing.T) {
	procs := readProcesses([]byte(`    1     0  10000
  100     1   1000
  101   100   2000
  102   101   4000
  200     1   8000
  bad line
`))
	for pid, want := range map[int]int64{100: 7000 << 10, 101: 6000 << 10, 200: 8000 << 10, 999: 0, 0: 0} {
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
