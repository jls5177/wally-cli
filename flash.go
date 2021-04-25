package wallycli

import (
	"errors"
	"fmt"
	"os"
)

type BoardFlashMethod func(firmwarePath string, s *FlashState) error

type BoardFlasher struct {
	*FlashState

	boardType BoardType
	boardFlashMethod BoardFlashMethod
}

func New(boardType BoardType) (*BoardFlasher, error) {
	flashRoutine, err := flashMethod(boardType)
	if err != nil {
		return nil, err
	}

	return &BoardFlasher{
		FlashState: newFlashState(),
		boardType: boardType,
		boardFlashMethod: flashRoutine,
	}, nil
}

type BoardType int

const (
	DfuBoard BoardType = iota
	TeensyBoard
)

type FlashStep int

const (
	Initializing FlashStep = iota
	InProgress
	Finished
)

type FlashState struct {
	step     FlashStep
	total    int
	sent     int
	flashErr error
}

func newFlashState() *FlashState {
	return &FlashState{
		step:  Initializing,
		total: 0,
		sent:  0,
	}
}

func (s *FlashState) Running() bool {
	return s.step > Initializing && s.step != Finished
}

func (s *FlashState) Finished() bool {
	return s.step == Finished
}

func (s *FlashState) TotalSteps() int {
	return s.total
}

func (s *FlashState) CompletedSteps() int {
	return s.sent
}

func (s *FlashState) FlashError() error {
	return s.flashErr
}

func flashMethod(boardType BoardType) (BoardFlashMethod, error) {
	var flashMethod BoardFlashMethod
	switch boardType {
	case DfuBoard:
		flashMethod = dfuFlash
	case TeensyBoard:
		flashMethod = teensyFlash
	default:
		return nil, fmt.Errorf("unsupported board type: %v", boardType)
	}
	return flashMethod, nil
}

func validateFilePath(path string) error {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("the file path you specified does not exist: %v", path)
	} else if err != nil {
		return fmt.Errorf("error parsing path: %w", err)
	}
	return nil
}

func (b *BoardFlasher) FlashAsync(path string) error {
	if err := validateFilePath(path); err != nil {
		return err
	}

	// call the flash method in the background to allow caller to
	go func() {
		_ = b.Flash(path)
	}()

	return nil
}

func (b *BoardFlasher) Flash(path string) error {
	if err := validateFilePath(path); err != nil {
		return err
	}
	b.flashErr = b.boardFlashMethod(path, b.FlashState)
	b.step = Finished
	return b.flashErr
}
