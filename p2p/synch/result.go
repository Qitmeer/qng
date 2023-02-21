package synch

type ProcessResultAction int

const (
	ProcessResultActionNull ProcessResultAction = iota
	ProcessResultActionContinue
	ProcessResultActionTryAgain
)

func (a ProcessResultAction) String() string {
	switch a {
	case ProcessResultActionNull:
		return "ProcessResultActionNull"
	case ProcessResultActionContinue:
		return "ProcessResultActionContinue"
	case ProcessResultActionTryAgain:
		return "ProcessResultActionTryAgain"
	}
	return ""
}

func (a ProcessResultAction) IsNothing() bool {
	return a == ProcessResultActionNull
}

func (a ProcessResultAction) IsContinue() bool {
	return a == ProcessResultActionContinue
}

func (a ProcessResultAction) IsTryAgain() bool {
	return a == ProcessResultActionTryAgain
}

type ProcessResult struct {
	act    ProcessResultAction
	orphan bool
	add    int
}

func NewProcessResult() *ProcessResult {
	return &ProcessResult{act: ProcessResultActionNull, orphan: false}
}
