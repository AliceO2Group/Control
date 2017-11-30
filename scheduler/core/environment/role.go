package environment

import "github.com/looplab/fsm"



// TODO: this is the FSM of each O² process, for further reference
//fsm := fsm.NewFSM(
//	"STANDBY",
//	fsm.Events{
//		{Name: "CONFIGURE", Src: []string{"STANDBY", "CONFIGURED"},           Dst: "CONFIGURED"},
//		{Name: "START",     Src: []string{"CONFIGURED"},                      Dst: "RUNNING"},
//		{Name: "STOP",      Src: []string{"RUNNING", "PAUSED"},               Dst: "CONFIGURED"},
//		{Name: "PAUSE",     Src: []string{"RUNNING"},                         Dst: "PAUSED"},
//		{Name: "RESUME",    Src: []string{"PAUSED"},                          Dst: "RUNNING"},
//		{Name: "EXIT",      Src: []string{"CONFIGURED", "STANDBY"},           Dst: "FINAL"},
//		{Name: "GO_ERROR",  Src: []string{"CONFIGURED", "RUNNING", "PAUSED"}, Dst: "ERROR"},
//		{Name: "RESET",     Src: []string{"ERROR"},                           Dst: "STANDBY"},
//	},
//	fsm.Callbacks{},
//)


type O2Process struct {
	Name		string			`json:"name" binding:"required"`
	Path		string			`json:"path" binding:"required"`
	Args		[]string		`json:"args" binding:"required"`
	Fsm			*fsm.FSM		`json:"-"`	// skip
	//			↑ this guy will initially only have 2 states: running and not running, or somesuch
	//			  ... or we could also do the same with NO state machine, and only add the state machine
	//			  as a meaningful entity for an already running FairMQ device. TBD.
}

func NewO2Process() *O2Process {
	return &O2Process{

	}
}

type RoleClass struct {
	className	string

}

type Role struct {
	Name		string			`json:"name" binding:"required"`
	Processes 	[]O2Process		`json:"processes" binding:"required"`
	RoleCPU		float64			`json:"roleCPU" binding:"required"`
	RoleMemory	float64			`json:"roleMemory" binding:"required"`
}