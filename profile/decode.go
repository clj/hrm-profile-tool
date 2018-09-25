package profile

import (
	"encoding/binary"
	"io"

	"github.com/clj/hrm-profile-tool/instructions"
)

const (
	FILE_HEADER_OFFSET = 0
	FILE_HEADER_SIZE   = 36
	FLOOR_HEADER_SIZE  = 40
	FLOOR_TAB_SIZE     = 46252
	INSTRUCTIONS_SIZE  = 4100
)

// Number of floors present in the save file
// Some floors are cut-scenes and are therefore not
// present
const numFloors = 36

// The 'missing' floors (i.e. cut-scenes)
var missingFloors = [...]int{5, 15, 18, 27, 33, -1}

// The order of the data in the profile is not exactly
// that of the order of floors in the game, this provides
// a mapping from floors to indexes into the profile data
// file
var floorToIdx = map[int]int{
	36: 39,
	37: 36,
	38: 40,
	39: 37,
	40: 41,
	41: 38,
}

// Provides the above mapping, but in reverse
var idxToFloor map[int]int // set up in init()

// Given a floor (as shown in the game) return the index
// in the profile data file for that floor
func FloorToIndex(floor int) int {
	if adjustedFloor, found := floorToIdx[floor]; found {
		floor = adjustedFloor
	}
	var missingFloor int
	i := 0
	for i, missingFloor = range missingFloors {
		if floor < missingFloor {
			break
		}
	}
	return floor - i - 1
}

// Given an index into the profile data file return the
// floor number (as shown in the game)
func IndexToFloor(index int) int {
	var missingFloor int
	i := 0
	for i, missingFloor = range missingFloors {
		if index < missingFloor-1-i {
			break
		}
	}
	floor := index + i + 1
	if adjustedFloor, found := idxToFloor[floor]; found {
		floor = adjustedFloor
	}
	return floor
}

// Given a profile number and a floor index (e.g. from FloorToIndex) return the start address
// in the profiles.bin file of the floor
func FloorStartAddr(profile, floorIndex int) int64 {
	return int64(FILE_HEADER_OFFSET + FILE_HEADER_SIZE + floorIndex*(FLOOR_HEADER_SIZE+FLOOR_TAB_SIZE*3))
}

// Given a profile number and a floor index (e.g. from FloorToIndex), and a tab number return
// the start address in the profiles.bin file of the tab from that floor
func TabStartAddr(profile, floorIndex, tab int) int64 {
	return FloorStartAddr(profile, floorIndex) + FLOOR_HEADER_SIZE + int64(FLOOR_TAB_SIZE*tab)
}

// A decoded code tab
type Tab struct {
	Offset      int
	Code        instructions.Disassembled
	RawComments instructions.RawComments
	Comments    instructions.Comments
}

// A decoded floor
type Floor struct {
	Offset         int
	Completed      bool
	SizeChallenge  int
	SpeedChallenge int
	Tabs           [3]Tab
}

// A decoded profile
type Profile struct {
	Floors [numFloors]Floor
}

// The raw floor header
type FloorHeader struct {
	Unknown0                uint32
	Unknown1                uint32
	Unknown2                uint32
	Unknown3                uint32
	SizeChallengeCompleted  int32
	SpeedChallengeCompleted int32
	SizeChallengeCommands   uint32
	SpeedChallengeSteps     uint32
	Unknown8                uint32
	Unknown9                uint32
}

// Given an in game floor number, return the floor data
func (p Profile) GetFloor(number int) Floor {
	return p.Floors[FloorToIndex(number)]
}

// func (t Tab) RenderSVG() string {
// 	return instructions.RenderSVG(t.Code, t.Comments)
// }

// func (t Tab) RenderText() string {
// 	str := instructions.RenderText(t.Code)
// 	if len(t.RawComments) > 0 {
// 		str += "\n\n" + wrap(instructions.RenderCommentsText(t.RawComments), 80)
// 	}
// 	return str
// }

// Decode and return a profile from the given reader
func Decode(reader io.ReadSeeker) (Profile, error) {
	var profile Profile

	missingIdx := 0
	for floorNumber := 0; floorNumber < numFloors; floorNumber++ {
		var floorHeader FloorHeader
		var floor Floor
		if missingIdx < len(missingFloors) && floorNumber+1+missingIdx == missingFloors[missingIdx] {
			missingIdx++
		}
		floor_start := int64(FILE_HEADER_OFFSET + FILE_HEADER_SIZE + floorNumber*(FLOOR_HEADER_SIZE+FLOOR_TAB_SIZE*3))
		reader.Seek(floor_start, io.SeekStart)
		if err := binary.Read(reader, binary.LittleEndian, &floorHeader); err != nil {
			return Profile{}, err
		}
		floor.SizeChallenge, floor.SpeedChallenge = -1, -1
		if floorHeader.SpeedChallengeCompleted > 0 {
			floor.SpeedChallenge = int(floorHeader.SpeedChallengeSteps)
		}
		if floorHeader.SizeChallengeCompleted > 0 {
			floor.SizeChallenge = int(floorHeader.SizeChallengeCommands)
		}

		for tab := 0; tab < 3; tab++ {
			tab_start := floor_start + FLOOR_HEADER_SIZE + int64(FLOOR_TAB_SIZE*tab)

			reader.Seek(tab_start, io.SeekStart)

			instructionList, err := instructions.DecodeInstructions(reader)
			if err != nil {
				return Profile{}, err
			}
			floor.Tabs[tab].Code = instructions.Disassemble(instructionList)
			if err != nil {
				return Profile{}, err
			}

			reader.Seek(tab_start+INSTRUCTIONS_SIZE, io.SeekStart)
			floor.Tabs[tab].RawComments, err = instructions.DecodeRawComments(reader)
			if err != nil {
				return Profile{}, err
			}
			floor.Tabs[tab].Comments, err = instructions.DecodeComments(floor.Tabs[tab].RawComments)
			if err != nil {
				return Profile{}, err
			}
		}
		profile.Floors[floorNumber] = floor
	}
	return profile, nil
}

func init() {
	idxToFloor = make(map[int]int)
	for key := range floorToIdx {
		idxToFloor[floorToIdx[key]] = key
	}
}
