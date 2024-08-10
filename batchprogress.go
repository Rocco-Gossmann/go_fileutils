package go_fileutils

type ProgressState int

const (
	STATE_START_FILE ProgressState = 1
	STATE_COPY       ProgressState = 2
	STATE_END_FILE   ProgressState = 3
	STATE_FINISHED   ProgressState = 4
	STATE_ERROR      ProgressState = 5
)

type BatchProgress struct {
	CurrentSource string
	CurrentTarget string
	State         ProgressState
	BytesTotal    int
	BytesCopied   int
	Error         error
}
