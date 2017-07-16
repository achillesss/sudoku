package main

import (
	"bufio"
	"flag"
	"fmt"
	"strconv"
	"strings"
	"time"

	"math"

	log "github.com/achillesss/log"
)

const full = 1<<9 - 1

type block struct {
	position int
	value    int
}

type region struct {
	blocks []*block
}

type sudoku struct {
	*region
}

// 行为各位，列为十位

// 111 111 111 = 0777
// 100 000 000 = 0400 9
// 010 000 000 = 0200 8
// 001 000 000 = 0100 7
// 000 100 000 = 0040 6
// 000 010 000 = 0020 5
// 000 001 000 = 0010 4
// 000 000 100 = 0004 3
// 000 000 010 = 0002 2
// 000 000 001 = 0001 1

// 将一个各位十进制数 n 转化成需要格式的二进制 m
// 该二进制数 m 表示为：将二进制1向右移动 n-1 位

func (b block) getRow() int {
	return b.position % 10
}

func (b block) getColumn() int {
	return b.position / 10
}

func (b block) getBoxPosition() int {
	return boxPosition(b.getRow(), b.getColumn())
}

func blockPosition(row, column int) int {
	return row + column*10
}

func boxPosition(row, column int) int {
	return (row-1)/3 + (column-1)/3*10
}

func (r *region) blockValue(position int) int {
	if b := r.getBlock(position); b != nil {
		return b.value
	}
	return 0
}

func (r *region) setValue(position, byteValue int) {
	if b := r.getBlock(position); b != nil {
		b.value = byteValue
	} else {
		newB := new(block)
		newB.position = position
		newB.value = byteValue
		r.blocks = append(r.blocks, newB)
	}
}

func (r *region) getBlock(position int) *block {
	for _, b := range r.blocks {
		if b.position == position {
			return b
		}
	}
	return nil
}

var testSudoku = []string{
	"008020090",
	"030007001",
	"000080070",
	"900704006",
	"003800200",
	"600300008",
	"060030000",
	"300500080",
	"050010600",
}

func convert(num int) int {
	return 1 << uint(num-1)
}

// 计算一个小块中有多少个可能值
func countFlag(num int) int {
	var n int
	for i := 0; i < 9; i++ {
		if num&1 == 1 {
			n++
		}
		num >>= 1
	}
	return n
}

func main() {
	flag.Parse()
	log.Infofln("start")

	// inputBufio := bufio.NewReader(os.Stdin)
	// initFunc := readBlock(inputBufio)
	var s sudoku
	s.region = new(region)

	for j := 0; j < 9; j++ {
		inputBufio := bufio.NewReader(strings.NewReader(testSudoku[j]))
		initFunc := s.readBlock(inputBufio)
		inputErr := initFunc(j + 1)
		for inputErr != nil {
			inputErr = initFunc(j + 1)
		}
	}

	s.draw("原始数据：", true)
	s.rejectOtherBlocks()
	s.tryRiddle()
	monitor()

}

// 输入一串数字字符串，输出一个块组函数
// 若某一小块不为0， 则该块为已知块，记录该块数字
// 若某一小块为0， 则暂时记录该值为所有可能出现的值
// 所有可能的值为输入中未出现的值

func blockIntersection(src, value int) int {
	if src == 0 {
		return value
	}
	return src & value
}

func (s sudoku) inputSudoku(input string) func(int) {
	var rest = full
	return func(row int) {
		for i, v := range input {
			p := blockPosition(row, i+1)
			v, _ := strconv.ParseInt(string(v), 10, 64)
			var byteValue int
			if v != 0 {
				byteValue = convert(int(v))
				rest = rest ^ byteValue
			}
			s.setValue(p, byteValue)
		}

		for _, b := range s.blocks {
			if b.value == 0 {
				s.setValue(b.position, rest)
			}
		}
	}
}

func convertInput(input string) (string, error) {
	input = strings.TrimSuffix(input, "\n")
	num, err := strconv.ParseInt(input, 10, 64)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%09d", num), nil
}

// 输入一个完整的数独
func (s sudoku) readBlock(r *bufio.Reader) func(int) error {
	return func(n int) error {
		log.Infofln("输入第%d行，回车确认：", n)

		data, _ := r.ReadString('\n')
		log.Infofln("输入：%s", data)
		newInput, err := convertInput(data)

		if err == nil {
			blockFunc := s.inputSudoku(newInput)
			blockFunc(n)
		}

		return err
	}
}

// // Brick划分：
// // 取 (x-1)/3+(y-1)/3*10 为brick_id
// // 00, 01, 02, 10, 11, 12, 20, 21, 22 九个brick

const (
	row = iota
	column
	box
)

func (b block) canBeExtracted(blocksType, blocksTypeID int) (ok bool) {
	switch blocksType {
	case box:
		ok = b.getBoxPosition() == blocksTypeID
	case row:
		ok = b.getRow() == blocksTypeID
	case column:
		ok = b.getColumn() == blocksTypeID
	}
	return
}

func extractReginBlocks(src region) func(blocksType, blocksTypeID int) (r *region) {
	return func(blocksType, blocksTypeID int) *region {
		r := new(region)
		for _, b := range src.blocks {
			if b.canBeExtracted(blocksType, blocksTypeID) {
				r.blocks = append(r.blocks, b)
			}
		}
		return r
	}
}

// toRealNum 输出原始数字
func (s region) draw(info string, toRealNum bool) {
	e := "%09b"
	if toRealNum {
		e = "%1d"
	}
	var cl, ro []string
	var rov []interface{}
	extFunc := extractReginBlocks(s)
	for i := 0; i < 9; i++ {
		bs := extFunc(row, i+1)
		subExtFunc := extractReginBlocks(*bs)
		ro = append(ro, e)
		for j := 0; j < 9; j++ {
			cb := subExtFunc(column, j+1)
			blockValue := cb.blockValue(blockPosition(i+1, j+1))
			if toRealNum {
				flagCount := countFlag(blockValue)
				pickFunc := pickBitValue(blockValue)
				flag := pickFunc(1)
				blockValue = int(math.Log2(float64(flag)) + 1)
				if flagCount > 1 {
					blockValue = 0
				}
			}
			rov = append(rov, blockValue)
		}
	}

	for i := 0; i < 9; i++ {
		cl = append(cl, strings.Join(ro, " | "))
	}

	formation := "%s\n九宫格：\n\t\t" + strings.Join(cl, "\n\t\t") + "\n"
	rov = append([]interface{}{info}, rov...)
	log.Infofln(formation, rov...)
}

func (r *region) blockRejector(rightNum, position int) (checkAgain bool) {
	v := r.blockValue(position)
	if v != rightNum {
		r.setValue(position, v&^rightNum)
	}
	checkAgain = countFlag(r.blockValue(position)) == 1 && countFlag(v) != 1
	return
}

func rejectOtherBlocksByBlock(r *region, b *block) (checkAgain bool) {
	if countFlag(b.value) == 1 {

		extFunc := extractReginBlocks(*r)

		rBlocks := extFunc(row, b.getRow())
		if rBlocks.regionRejector(b.value) {
			checkAgain = true
		}
		// r.draw(fmt.Sprintf("根据%+v剔除行", b))

		extFunc = extractReginBlocks(*r)
		cBlocks := extFunc(column, b.getColumn())
		if cBlocks.regionRejector(b.value) {
			checkAgain = true
		}
		// r.draw(fmt.Sprintf("根据%+v剔除列", b))

		extFunc = extractReginBlocks(*r)
		bBlocks := extFunc(box, int(b.getBoxPosition()))
		if bBlocks.regionRejector(b.value) {
			checkAgain = true
		}
		// r.draw(fmt.Sprintf("根据%+v剔除宫", b))

	}

	if countFlag(b.value) == 2 {
		extFunc := extractReginBlocks(*r)
		rBlocks := extFunc(row, b.getRow())
		if rBlocks.multiRejection() {
			checkAgain = true
		}

		extFunc = extractReginBlocks(*r)
		cBlocks := extFunc(column, b.getColumn())
		if cBlocks.multiRejection() {
			checkAgain = true
		}

	}
	return
}
func (r *region) rejectOtherBlocks() {
	var checkAgain bool
	for _, b := range r.blocks {
		if rejectOtherBlocksByBlock(r, b) {
			checkAgain = true
		}
	}
	if checkAgain {
		r.rejectOtherBlocks()
	}
}

func (r *region) regionRejector(rightNum int) (checkAgain bool) {
	for _, b := range r.blocks {
		if r.blockRejector(rightNum, b.position) {
			checkAgain = true
		}
	}
	return
}

func (r region) finished() int {
	var total int
	var maxFlag int
	var minFlag = 9
	var status int

	for _, b := range r.blocks {
		maxFlag = int(math.Max(float64(maxFlag), float64(countFlag(b.value))))
		minFlag = int(math.Min(float64(minFlag), float64(countFlag(b.value))))
		total |= b.value
	}

	if maxFlag == 1 && total == full {
		status = 1
	} else if maxFlag == 1 {
		status = 2
	} else if minFlag == 0 {
		status = 3
	} else {
		status = 0
	}
	return status
}

func (s sudoku) check() int {
	status := 1
	extFunc := extractReginBlocks(*s.region)
	for i := 0; i < 9; i++ {
		r := extFunc(row, i+1)
		c := extFunc(column, i+1)

		if r.finished() == 2 || c.finished() == 2 {
			return 2
		}

		if r.finished() == 3 || c.finished() == 3 {
			return 3
		}

		if r.finished() == 0 || c.finished() == 0 {
			status = 0
		}

		for j := 0; j < 3; j++ {
			if i%3 == 0 {
				b := extFunc(box, boxPosition(i/3, j))
				if b.finished() == 2 {
					return 2
				}

				if b.finished() == 3 {
					return 3
				}
				if b.finished() == 0 {
					status = 0
				}
			}
		}
	}
	return status
}

func (s *sudoku) copy() sudoku {
	var newS sudoku
	newS.region = new(region)
	for _, b := range s.blocks {
		newB := new(block)
		*newB = *b
		newS.blocks = append(newS.blocks, newB)
	}
	return newS
}

func (r *region) multiRejection() (checkAgain bool) {
	valuePositionMap := make(map[int][]int)
	for _, b := range r.blocks {
		valuePositionMap[b.value] = append(valuePositionMap[b.value], b.position)
	}

	for k, v := range valuePositionMap {
		if countFlag(k) == len(v) {
			for _, b := range r.blocks {
				if b.value != k {
					r.setValue(b.position, b.value&^k)
					if countFlag(b.value) != 1 && countFlag(r.blockValue(b.position)) == 1 {
						checkAgain = true
					}
				}
			}
		}
	}

	return
}

type sudokuStauts struct {
	status    int
	s         sudoku
	cost      float64
	position  int
	preValue  int
	postValue int
}

var statusChan = make(chan *sudokuStauts, 100)

func (s *sudoku) tryRiddle() {
	start := time.Now()
	for _, b := range s.blocks {
		flagCount := countFlag(b.value)
		pickFunc := pickBitValue(b.value)
		go func(b *block) {
			if flagCount > 1 {
				for j := 0; j < flagCount; j++ {
					go func(j int, b *block) {
						newSudoku := s.copy()
						v := pickFunc(j + 1)
						newSudoku.setValue(b.position, v)
						newSudoku.rejectOtherBlocks()
						status := newSudoku.check()
						statusChan <- &sudokuStauts{status, newSudoku, time.Now().Sub(start).Seconds() * 1000, b.position, b.value, v}
					}(j, b)
				}
			}
		}(b)

	}
}

func monitor() {
	for status := range statusChan {
		if status.status == 1 {
			status.s.draw(fmt.Sprintf("解块%d，用%d替换%d，成功，用时%.2fms", status.position, status.postValue, status.preValue, status.cost), true)
		}
		if status.status == 0 {
			// status.s.draw(fmt.Sprintf("解块%d，用%d替换%d，失败，用时%.2fms", status.position, status.postValue, status.preValue, status.cost))
			status.s.tryRiddle()
		}
	}
}

func pickBitValue(byteValue int) func(order int) int {
	return func(order int) int {
		flag := 1
		for i := 0; i < order; i++ {
			for byteValue&flag == 0 {
				flag <<= 1
			}
			flag <<= 1
		}
		return flag >> 1
	}
}
