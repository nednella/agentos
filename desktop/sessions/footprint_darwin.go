package sessions

/*
#include <libproc.h>
#include <sys/resource.h>
*/
import "C"

import "unsafe"

// footprint is the memory a process uses as Activity Monitor counts it: its own resident and
// compressed pages, without the libraries it shares. RSS leaves out compressed pages, which are
// most of what a process holds once the Mac is short of memory. 0 when the process is gone.
func footprint(pid int) int64 {
	var info C.struct_rusage_info_v0
	if C.proc_pid_rusage(C.int(pid), C.RUSAGE_INFO_V0, (*C.rusage_info_t)(unsafe.Pointer(&info))) != 0 {
		return 0
	}
	return int64(info.ri_phys_footprint)
}
