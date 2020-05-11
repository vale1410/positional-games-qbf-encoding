package main

import (
	"bufio"
	"errors"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	flag "github.com/spf13/pflag"
)

var encFlag int
var depFlag bool

var c10Flag bool
var c16Flag bool
var c18Flag bool
var c19Flag bool
var symFlag bool

var moveFlag bool
var stackFlag bool

var connectFlag bool

var cntFlag bool
var cntThresh int

var fixFlag bool
var mbFlag bool
var bwinFlag bool
var amoFlag bool // redundant constraint, at most one move per position for player black!
var swpFlag bool
var insFlag bool

func initLogic() {
	flag.IntVarP(&encFlag, "enc", "1", 1, "0: set all deprecated flags on. 1: c16. 2. c10,c18,c19. 3. 4. 5. ")
	flag.BoolVarP(&c10Flag, "c10", "", false, "adds redundant clause extrcd inst	a c10.")
	flag.BoolVarP(&c16Flag, "c16", "", false, "adds clause c16 (alternative c19).")
	flag.BoolVarP(&c18Flag, "c18", "", false, "adds redundant clause c18.")
	flag.BoolVarP(&c19Flag, "c19", "", false, "adds clause c19 (alternative c16).")
	flag.BoolVarP(&symFlag, "sym", "", true, "adds symmetry clauses.")
	flag.BoolVarP(&stackFlag, "stack", "s", false, "Adds Stack into MB encoding.")
	flag.BoolVarP(&moveFlag, "move", "m", false, "Redundant definition of move() to MB encoding. \n  move(B,P,J) <=> ~board(B,P,I), board(B,P,J), succ(I,J).")
	flag.BoolVarP(&connectFlag, "connect", "", false, "Adds Clauses for ConnectX logic if vertical columns.")
	cntThresh = 6 // cnt Threshold

	flag.Parse()
	False = "F"

	switch encFlag {
	case 1: // MM
		c16Flag = true
	case 2: // MM+
		c10Flag = true
		c18Flag = true
		c19Flag = true
	case 3: // MM DUPER
	case 4: // MB encoding
		cntFlag = true
		stackFlag = true
		moveFlag = false
	case 5: // MM DUPER INCREMENTAL STRAIGHT
	case 6: // MM DUPER INCREMENTAL LOGARITHMIC
	default:
		panic("please choose encoding by --enc=<id>" + ". Chosen:" + fmt.Sprintf("%v", encFlag))
	}
}

func main() {
	initLogic()
	g, err := parse()
	if err != nil {
		fmt.Println("error parsing file:")
		fmt.Println(err)
		os.Exit(1)
	}

	// Transforms multiple consecutive moves by same player into one move with number of steps
	// notation-wise the connected moves are just concatenated, i.e. a1 and a2 -> a1a2
	g.translateToSimultaneousMoves()

	switch encFlag {
	case 1:
		g.encodeMakerMaker()
	case 2:
		g.encodeMakerMaker()
	case 3:
		g.encodeMakerMakerEmove()
	case 4:
		if len(g.whitewins) != 0 || len(g.firstmoves) != 0 {
			panic("is not a MakerBreaker instance!!!")
		}
		g.encodeMakerBreaker()
	case 5:
		g.encodeMakerMakerLogarithmic()
	default:
		g.encodeMakerMaker5()
	case 6:
		g.generateBuleFacts()
	case 7:
		panic("please choose encoding by --enc=<id>")
	}
}

// LOGIC STUFF
var False string
var cls clauses

type clause []string
type clauses struct {
	cs []clause
}

func (cls *clauses) addCls(cl clause) {

	cl2 := make([]string, 0, len(cl))
	for _, c := range cl {
		if c == neg(False) {
			return
		} else if c != False {
			cl2 = append(cl2, c)
		}
	}
	if len(cl2) == 0 {
		fmt.Println(cl)
		panic("Empty clause generated")
	}
	cls.cs = append(cls.cs, cl2)
	return
}

func (cls *clauses) add(cl ...string) {
	cls.addCls(clause(cl))
}

func (cls *clauses) comment(s string) {
	cls.cs = append(cls.cs, []string{"c", s})
}

//s <=> OR(i)
func (cls *clauses) addEquivalenceDisjunction(s string, disjunction []string) {
	cls.addCls(append(clause{neg(s)}, disjunction...))
	for _, x := range disjunction {
		cls.add(s, neg(x))
	}
}

////s <=> AND(i)
//func (cls *clauses) addEquivalenceConjunction(s string, conjunction []string) {
//	cl := clause{s}
//	for _, x := range conjunction {
//		cl = append(cl, neg(x))
//		cls.add(neg(s), x)
//	}
//	cls.addCls(cl)
//}

type game struct {
	positions []string
	blackwins [][]string
	whitewins [][]string

	times         []string
	blackturns    []string
	whiteturns    []string
	blackinitials []string
	whiteinitials []string
	firstmoves    []string

	numbers     map[string]int // mapping of time_id -> to number of moves
	connections [][]string     // list of tuples of positions
}

// Transforms the game description to go from multiple consecutive moves for one player
// to only one move with multiple moves (stored in numbers).
func (g *game) translateToSimultaneousMoves() {

	var times []string
	var blackturns []string
	var whiteturns []string
	numbers := map[string]int{} // number of moves within one time point

	{ // put format into simultaneous moves
		// contract the time steps into one if consecutive

		ib := 0   // black pos in array
		iw := 0   // white pos in array
		idw := "" // concatenated white ids
		idb := "" // concatenated black ids
		n := 0

		for _, x := range g.times {
			if len(g.blackturns) > 0 && x == g.blackturns[ib] {
				if idw != "" {
					whiteturns = append(whiteturns, idw)
					times = append(times, idw)
					numbers[idw] = n
					idw = ""
					n = 0
				}

				if ib < len(g.blackturns)-1 {
					ib++
				}
				// connect black ids
				idb += x
				n++
			} else if len(g.whiteturns) > 0 && x == g.whiteturns[iw] {

				if idb != "" {
					blackturns = append(blackturns, idb)
					times = append(times, idb)
					numbers[idb] = n
					idb = ""
					n = 0
				}

				if iw < len(g.whiteturns)-1 {
					iw++
				}
				// connect white ids
				idw += x
				n++
			} else {
				panic("wrong order")
			}
		}
		if idb != "" {
			blackturns = append(blackturns, idb)
			times = append(times, idb)
			numbers[idb] = n
		}
		if idw != "" {
			whiteturns = append(whiteturns, idw)
			times = append(times, idw)
			numbers[idw] = n
		}
	}
	g.times = times
	g.blackturns = blackturns
	g.whiteturns = whiteturns
	g.numbers = numbers
}

// Maker/Maker Encoding

func (g *game) encodeMakerMaker() {

	times := g.times
	blackturns := g.blackturns
	whiteturns := g.whiteturns
	numbers := g.numbers
	positions := g.positions
	whitewins := g.whitewins
	blackwins := g.blackwins
	firstmoves := g.firstmoves

	cheat := "cheat"
	win := "win"
	lose := "lose"
	black := "B"
	white := "W"

	succ := map[string]string{}
	for i, t := range times {
		if i == len(times)-1 {
			succ[t] = False
		} else {
			succ[t] = times[i+1]
		}
	}

	cls = clauses{cs: []clause{}}
	quantifier := []string{}

	// MAKER MAKER GENERAL ENCODING
	{ // Create quantifiers
		bi := 0
		wi := 0

		for _, t := range times {
			quantifier = append(quantifier, "e "+time(t))

			var s string
			var player string
			if bi < len(blackturns) && blackturns[bi] == t {
				s = "e"
				player = black
				bi++
			}
			if wi < len(whiteturns) && whiteturns[wi] == t {
				s = "a"
				player = white
				wi++
			}
			for _, p := range positions {
				s += " " + move(player, p, t)
			}
			quantifier = append(quantifier, s)

			if in(t, whiteturns) {
				s = "e"
				subsets := allSubsetsSize(positions, numbers[t]+1)
				for _, subset := range subsets {
					s += " " + excess(subset, t)
				}
				if t != times[0] {
					for _, p := range positions {
						s += " " + stack(p, t)
					}
				}
				quantifier = append(quantifier, s)
			}
		}

		{
			s := "e " + win + " " + lose
			for _, p := range positions {
				s += " " + final(black, p)
				s += " " + final(white, p)
			}
			for i, _ := range blackwins {
				s += " " + win_row_s(i)
			}
			quantifier = append(quantifier, s)
		}

		if wi != len(whiteturns) || bi != len(blackturns) {
			fmt.Println("whiteturns", whiteturns)
			fmt.Println("blackturns", blackturns)
			panic("inconsistent game description: whiteturns union blackturns = times")
		}
	}

	{ // (3)
		cls.comment("3\t: time(T) -> time(T+1)")
		for _, t := range times {
			cls.add(time(t), neg(time(succ[t])))
		}
	}

	{ // (4)
		cls.comment("4\t: move(black,P,T) -> time(T)")
		for _, t := range blackturns {
			for _, p := range positions {
				cls.add(time(t), neg(move(black, p, t)))
			}
		}
	}

	{ // (5)
		cls.comment("5\t: ~final(black,P), move(black,P,T) : time(T)")
		for _, p := range positions {
			c := clause{neg(final(black, p))}
			for _, t := range blackturns {
				c = append(c, move(black, p, t))
			}
			cls.addCls(c)
		}
	}

	{ // (6)
		cls.comment("6\t: final(white,P), ~time(T), ~move(white,P,T)")
		for _, t := range whiteturns {
			for _, p := range positions {
				cls.add(final(white, p), neg(time(t)), neg(move(white, p, t)))
			}
		}
	}

	{ // (7) (8)
		cls.comment("7\t: cheat, win")
		cls.comment("8\t: cheat, ~lose")

		cls.add(cheat, win)
		cls.add(cheat, neg(lose))
	}

	{ // (9) (10) (10')
		cls.comment("9\t: win -> win(E) : edge(black,E)")
		cls.comment("10\t: in(P,E), win(E) -> final(black,P)")
		cls.comment("10b\t: final(black,P):sub(P,e) -> win(e)")

		b_win := clause{neg(win)}
		for i, ps := range blackwins {
			row := win_row_s(i)
			b_win = append(b_win, row)
			winImpl := clause{row}
			for _, p := range ps {
				cls.add(neg(row), final(black, p)) // (10)
				winImpl = append(winImpl, neg(final(black, p)))
			}
			if c10Flag {
				cls.addCls(winImpl) // (10')
			}
		}
		cls.addCls(b_win) // (9)
	}

	{ // (11)
		cls.comment("11\t: edge(white,E): lose \\/ ~final(white,P) : in(P,E)")
		for _, ps := range whitewins {
			c := clause{lose}
			for _, p := range ps {
				c = append(c, neg(final(white, p)))
			}
			cls.addCls(c)
		}
	}

	{
		cls.comment("12\t: ~move(black,P,T) : subset(P,N+1,positions)")
		for _, t := range blackturns {
			subsets := allSubsetsSize(positions, numbers[t]+1)
			for _, subset := range subsets {
				c := clause{}
				for _, p := range subset {
					c = append(c, neg(move(black, p, t)))
				}
				cls.addCls(c)
			}
		}
	}

	{
		cls.comment("13\t: I < J : ~move(white,P,T), ~move(black,P,J).")
		for i := len(times) - 1; i > 0; i-- {
			if in(times[i], blackturns) {
				for j := i - 1; j >= 0; j-- {
					if in(times[j], whiteturns) {
						for _, p := range positions {
							cls.add(neg(move(black, p, times[i])), neg(move(white, p, times[j])))
						}
					}
				}
			}
		}
	}

	{
		cls.comment("14\t: subset(P,S,N+1,positions), excess(S,T) -> move(white,P,T) ")
		for _, t := range whiteturns {
			subsets := allSubsetsSize(positions, numbers[t]+1)
			for _, subset := range subsets {
				for _, p := range subset {
					cls.add(neg(excess(subset, t)), move(white, p, t))
				}
			}
		}
	}

	{
		cls.comment("15\t: stack(P,T), -> move(white,P,T) ")
		for _, t := range whiteturns {
			if t != times[0] {
				for _, p := range positions {
					cls.add(neg(stack(p, t)), move(white, p, t))
				}

			}
		}
	}

	if c16Flag {
		cls.comment("16\t: stack(P,I) -> move(black,P,J) : I < J.")
		for _, p := range positions {
			for i := len(times) - 1; i > 0; i-- {
				if in(times[i], whiteturns) {
					cl := clause{neg(stack(p, times[i]))}
					for j := i - 1; j >= 0; j-- {
						if in(times[j], blackturns) {
							cl = append(cl, move(black, p, times[j]))
						}
					}
					cls.addCls(cl)
				}
			}
		}
	}

	if c19Flag {
		cls.comment("19\t: stack(P,I) -> move(white/black,P,J) : I < J.")
		for _, p := range positions {
			for i := len(times) - 1; i > 0; i-- {
				if in(times[i], whiteturns) {
					cl := clause{neg(stack(p, times[i]))}
					for j := i - 1; j >= 0; j-- {
						if in(times[j], blackturns) {
							cl = append(cl, move(black, p, times[j]))
						}
						if in(times[j], whiteturns) {
							cl = append(cl, move(white, p, times[j]))
						}
					}
					cls.addCls(cl)
				}
			}
		}
	}

	litWhiteCheatSymmetric := "cheatSymmetric"
	addToCheatClause := false
	if symFlag { // (symmetry) by first moves
		cls.comment("sym1\t: move(black,P,0) : firstMove(P).")
		cls.comment("sym2\t: firstMove(P), cheatSymmtry -> ~move(white,P,0).")
		if len(times) > 0 && len(firstmoves) > 0 {
			firstTime := times[0]
			if len(blackturns) > 0 && firstTime == blackturns[0] {
				c := clause{}
				for _, p := range firstmoves {
					c = append(c, move(black, p, firstTime))
				}
				cls.addCls(c)
			} else if len(whiteturns) > 0 && firstTime == whiteturns[0] {
				for _, p := range firstmoves {
					cls.add(neg(litWhiteCheatSymmetric), neg(move(white, p, firstTime)))
					addToCheatClause = true
				}
			}
		}
	}

	{
		cls.comment("sym3\t: cheat -> stack(P,T) : P: T, excess(S,T):S:T, cheatSymmetry.")
		cl := clause{neg(cheat)}
		for _, t := range whiteturns {
			if t != times[0] {
				for _, p := range positions {
					cl = append(cl, stack(p, t))
				}
			}
		}
		for _, t := range whiteturns {
			subsets := allSubsetsSize(positions, numbers[t]+1)
			for _, subset := range subsets {
				cl = append(cl, excess(subset, t))
			}
		}
		if addToCheatClause {
			cl = append(cl, litWhiteCheatSymmetric)
			quantifier[2] += " " + litWhiteCheatSymmetric
		}
		cls.addCls(cl)
	}

	if c18Flag {
		cls.comment("18\t: I < J : ~move(black,P,I), ~move(black,P,J)")
		for _, p := range positions {
			for i := 0; i < len(blackturns)-1; i++ {
				for j := i + 1; j < len(blackturns); j++ {
					cls.add(neg(move(black, p, blackturns[i])), neg(move(black, p, blackturns[j])))
				}
			}
		}
	}

	{ //OUTPUT Clauses
		for _, q := range quantifier {
			fmt.Println(q)
		}

		for _, clause := range cls.cs {
			for _, c := range clause {
				fmt.Print(c, " ")
			}
			fmt.Println()
		}
	}
}

// Maker/Maker Encoding2
// contains board and move variables
// uses counter encoding for counting moves
func (g *game) encodeMakerMakerEmove() {

	times := g.times
	blackturns := g.blackturns
	whiteturns := g.whiteturns
	numbers := g.numbers
	positions := g.positions
	//connections := g.connections TODO FIX CONNECTIONS

	black := "black"
	white := "white"
	players := []string{black, white}
	turn := map[string][]string{}
	turn[black] = g.blackturns
	turn[white] = g.whiteturns

	wins := map[string][][]string{}
	wins[black] = g.blackwins
	wins[white] = g.whitewins

	timesZero := append([]string{"t0_"}, times...)
	positionsZero := append([]string{"p0_"}, positions...)

	succ := map[string]string{}
	for i, t := range timesZero {
		if i == len(timesZero)-1 {
			succ[t] = False
		} else {
			succ[t] = timesZero[i+1]
		}
	}

	pred := map[string]string{}
	for i, t := range timesZero {
		if i == 0 {
			pred[t] = neg(False)
		} else {
			pred[t] = timesZero[i-1]
		}
	}

	cls = clauses{cs: []clause{}}
	quantifier := []string{}

	{ // Create quantifiers

		cls.comment("1a\t: time(T), #exist(T).")
		cls.comment("1b\t: move(black,_,T), #exist(T).")
		cls.comment("1c\t: move(white,_,T), #forall(T).")
		cls.comment("1d\t: board(_,_,T), #exist(T).")
		cls.comment("1e\t: count(_,_,_,T), #exist(T).")
		cls.comment("1e\t: occupied(_,T), #exist(T),")
		cls.comment("1f\t: step(_,_,_,T), #exist(T)")
		cls.comment("1g\t: emove(_,_,T), #exist(T).")
		cls.comment("everything else (win(_), win_row(_,_) are innermost.")

		bi := 0
		wi := 0

		for i, t := range timesZero {
			quantifier = append(quantifier, "e "+time(t))

			var s string
			var player string

			if i > 0 {
				if bi < len(blackturns) && blackturns[bi] == t {
					s = "e"
					player = black
					bi++
				}
				if wi < len(whiteturns) && whiteturns[wi] == t {
					s = "a"
					player = white
					wi++
				}
				for _, p := range positions {
					s += " " + move(player, p, t)
				}
				quantifier = append(quantifier, s)
			}

			{
				s = "e "
				for _, V := range positions {
					s += " " + occupied(V, t)
					for _, P := range players {
						s += " " + board(P, V, t)
					}
				}
				quantifier = append(quantifier, s)
			}

			if i > 0 {
				{
					s = "e "
					for I := 0; I <= numbers[t]+1; I++ {
						for _, V := range positionsZero {
							s += " " + count(player, V, I, t)
						}
					}
					quantifier = append(quantifier, s)
				}

				if in(t, whiteturns) {
					{
						s = "e"
						for _, v := range positions {
							s += " " + emove(player, v, t)
							for j := 1; j <= numbers[t]; j++ {
								s += " " + step(player, v, j, t)
							}
						}
						quantifier = append(quantifier, s)
					}
				}
			}
		}

		{
			s := "e "
			for _, P := range players {
				s += " " + win(P)
				for i, _ := range wins[P] {
					s += " " + win_row(P, i)
				}
			}
			quantifier = append(quantifier, s)
		}

		if wi != len(whiteturns) || bi != len(blackturns) {
			fmt.Println("whiteturns", whiteturns)
			fmt.Println("blackturns", blackturns)
			panic("inconsistent game description: whiteturns union blackturns = times")
		}
	}

	{
		cls.comment("1\t: ~time(T), time(T-1).")
		for _, t := range times {
			cls.add(neg(time(t)), time(pred[t]))
		}
		cls.add(time(timesZero[0]))
	}

	{
		cls.comment("2\t: ~board(P,V,T) , board(P,V,T+1).")
		for _, p := range players {
			for _, t := range times {
				for _, v := range positions {
					cls.add(neg(board(p, v, pred[t])), board(p, v, t))
				}
			}
		}
	}

	{
		cls.comment("3\t: ~board(b,V,T) , ~board(w,V,T).")
		for _, t := range times {
			for _, v := range positions {
				cls.add(neg(board(black, v, t)), neg(board(white, v, t)))
			}
		}
	}

	{
		cls.comment("4\t: occupied(V,T) <=> board(b,V,T) | board(w,V,T).")
		for _, t := range timesZero {
			for _, p := range positions {
				cls.addEquivalenceDisjunction(occupied(p, t), []string{board(black, p, t), board(white, p, t)})
			}
		}
	}

	{
		cls.comment("5\t: time(T), board(P,V,T-1) , ~board(P,V,T).")
		for _, P := range players {
			for _, T := range times {
				for _, V := range positions {
					cls.add(time(T), board(P, V, pred[T]), neg(board(P, V, T)))
				}
			}
		}
	}

	//{ TODO FIX CONNECTION
	//{
	//cls.comment("6\t: move(P,V,T+1), board(P,V,T) , ~board(P,V,T+1).")
	//for _, P := range players {
	//	for _, T := range turn[P] {
	//		for _, V := range positions {
	//			cls.add(move(P, V, T), board(P, V, pred[T]), neg(board(P, V, T)))
	//		}
	//	}
	//}
	//}

	//{ TODO FIX CONNECTION
	//	cls.comment("7\t: prerequisite(V,W): ~occupied(V,T-1) -> ~occupied(W,T).")
	//	for _, T := range timesZero {
	//		for _, pp := range connections {
	//			if len(pp) != 2 {
	//				panic("wrong size of tuples in connections.")
	//			}
	//			cls.add(occupied(pp[0], pred[T]), neg(occupied(pp[1], T)))
	//		}
	//	}
	//}

	{
		cls.comment("8 \t: ~win(white).")
		cls.comment("9 \t: win(black).")
		cls.comment("10 \t: win(P) <=> win(P,E) : edge(P,E).")
		cls.comment("11 \t: ~win(P,E) <=> ~board(P,V,F) : in(V,E,P).")

		cls.add(win(black))
		cls.add(neg(win(white)))
		Final := times[len(times)-1]
		{
			for _, P := range players {
				edges := []string{}
				for i, ps := range wins[P] {
					row := win_row(P, i)
					edges = append(edges, row)
					ins := []string{}
					for _, V := range ps {
						ins = append(ins, neg(board(P, V, Final)))
					}
					cls.addEquivalenceDisjunction(neg(row), ins)
				}
				cls.addEquivalenceDisjunction(win(P), edges)
			}
		}

	}

	{
		cls.comment("12\t: ~count(P,V-1,I,T), count(P,V,I,T).")
		cls.comment("13\t: ~count(P,V-1,I+1,T), count(P,V,I,T).")
		cls.comment("14\t: ~move(P,V-1,T), ~count(P,V-1,I,T), count(P,V,I+1,T).")
		cls.comment("15\t: move(P,V,T), ~count(P,V,I,T), count(P,V-1,I,T).")
		for _, P := range players {
			for _, T := range turn[P] {
				for i, V := range positions {
					var top int
					if P == black {
						top = numbers[T] + 1
					} else { // P == white
						top = numbers[T]
					}
					for I := 0; I <= top; I++ {
						V1 := positionsZero[i] // V1 = V - 1
						cls.add(neg(count(P, V1, I, T)), count(P, V, I, T))
						cls.add(move(P, V, T), neg(count(P, V, I, T)), count(P, V1, I, T))
						if I != top {
							cls.add(neg(count(P, V, I+1, T)), count(P, V1, I, T))
							cls.add(neg(move(P, V, T)), neg(count(P, V1, I, T)), count(P, V, I+1, T))
						}
					}
				}
			}
		}

		cls.comment("17\t: count(P,0,0,T).")
		cls.comment("18\t: ~count(P,0,1,T).")
		for _, P := range players {
			for _, T := range turn[P] {
				V0 := positionsZero[0]
				cls.add(count(P, V0, 0, T))
				cls.add(neg(count(P, V0, 1, T)))
			}
			if P == black {

			}

		}
	}

	{
		cls.comment("19\t: ~time(T), count(black,|V|, N_t,T).")
		cls.comment("20\t: ~count(black,|V|, N_t+1,T).")
		cls.comment("21\t: ~move(black,V,T), time(T).")
		cls.comment("22\t: ~move(black,V,T-1), ~occupied(V,T-1).")
		cls.comment("23\t: ~move(black,V,T), board(black,V,T).")
		for _, T := range turn[black] {
			cls.add(neg(time(T)), count(black, positions[len(positions)-1], numbers[T], T))
			cls.add(neg(count(black, positions[len(positions)-1], numbers[T]+1, T)))
			for _, V := range positions {
				cls.add(neg(move(black, V, T)), time(T))
				cls.add(neg(move(black, V, T)), neg(occupied(V, pred[T])))
				cls.add(neg(move(black, V, T)), board(black, V, T))
			}
		}

		{
			cls.comment("6a\t: P=black: move(P,V,T+1), board(P,V,T) , ~board(P,V,T+1).")
			cls.comment("6c\t: PP=white: board(PP,V,T) , ~board(PP,V,T+1). during black moves, white won't take")
			P := black
			PP := white
			for _, T := range turn[P] {
				for _, V := range positions {
					cls.add(move(P, V, T), board(P, V, pred[T]), neg(board(P, V, T)))
					cls.add(board(PP, V, pred[T]), neg(board(PP, V, T)))
				}
			}
		}
	}

	{

		{
			cls.comment("24\t: X=white: emove(X,V,T) <=> step(X,V,I,T) : I = 1..J : numbers(T,J).")
			cls.comment("25\t: X=white: ~step(X,V,I,T) <=> ~count(X,V,I,T) | count(X,V-1,I,T).")
			X := white
			for _, T := range turn[X] {
				for i, V := range positions {
					nums := make([]string, 0)
					for I := 1; I <= numbers[T]; I++ {
						V1 := positionsZero[i] // V1 = V - 1
						cls.addEquivalenceDisjunction(neg(step(X, V, I, T)), []string{neg(count(X, V, I, T)), count(X, V1, I, T)})
						if I > 0 {
							nums = append(nums, step(X, V, I, T))
						}
					}
					cls.addEquivalenceDisjunction(emove(X, V, T), nums)
				}
			}
		}

		{
			//			cls.comment("26a\t: pre(W,V): ~time(T), ~occupied(V,T-1), occupied(W,T-1), ~emove(white,V,T), board(white,V,T).")
			cls.comment("26b\t: ~time(T), occupied(V,T-1), ~emove(white,V,T), board(white,V,T).")
			for _, T := range turn[white] {

				for _, V := range positions {
					cl := clause{neg(time(T)), occupied(V, pred[T]), neg(emove(white, V, T)), board(white, V, T)}
					//for _, pp := range connections { TODO FIX CONNECTION
					//	if pp[1] == V {
					//		cl = append(cl, occupied(pp[0], pred[T]))
					//	}
					//}
					cls.addCls(cl)
				}
			}
		}

		{
			cls.comment("6b\t: P=white: emove(P,V,T), board(P,V,T-1) , ~board(P,V,T).")
			cls.comment("6d\t: PP=black: board(PP,V,T) , ~board(PP,V,T+1). during white moves, black won't take")
			P := white
			PP := black
			for _, T := range turn[P] {
				for _, V := range positions {
					cls.add(emove(P, V, T), board(P, V, pred[T]), neg(board(P, V, T)))
					cls.add(board(PP, V, pred[T]), neg(board(PP, V, T)))
				}
			}
		}

	}
	{
		cls.comment("27\t: ~occupied(V,0).")
		for _, V := range positions {
			cls.add(neg(occupied(V, timesZero[0])))
		}
	}

	if symFlag {
		cls.comment("28\t: P=black: move(black,V,1) : firstmoves(V). % break symmetries")
		T := times[0]
		if in(T, blackturns) && len(g.firstmoves) > 0 {
			cl := clause{}
			for _, V := range g.firstmoves {
				cl = append(cl, move(black, V, T))
			}
			cls.addCls(cl)
		}
	}

	{ //OUTPUT Clauses
		for _, q := range quantifier {
			fmt.Println(q)
		}

		for _, clause := range cls.cs {
			for _, c := range clause {
				fmt.Print(c, " ")
			}
			fmt.Println()
		}
	}
}

// ENCODING 5
func (g *game) encodeMakerMakerLogarithmic() {

	times := g.times
	blackturns := g.blackturns
	whiteturns := g.whiteturns
	numbers := g.numbers
	positions := g.positions

	black := "black"
	white := "white"
	opponent := map[string]string{black: white, white: black}
	players := []string{black, white}
	turn := map[string][]string{}
	turn[black] = g.blackturns
	turn[white] = g.whiteturns

	wins := map[string][][]string{}
	wins[black] = g.blackwins
	wins[white] = g.whitewins

	timesZero := append([]string{"t0_"}, times...)
	positionsZero := append([]string{"p0_"}, positions...)

	bitsPositions := int(math.Ceil(math.Log2(float64(len(positions)))))

	succ := map[string]string{}
	for i, t := range timesZero {
		if i == len(timesZero)-1 {
			succ[t] = False
		} else {
			succ[t] = timesZero[i+1]
		}
	}

	pred := map[string]string{}
	for i, t := range timesZero {
		if i == 0 {
			pred[t] = neg(False)
		} else {
			pred[t] = timesZero[i-1]
		}
	}

	cls = clauses{cs: []clause{}}
	quantifier := []string{}

	{ // Create quantifiers

		cls.comment("1a\t: time(T), #exist(T).")
		cls.comment("1b\t: move(black,V,T), #exist(T).")
		cls.comment("1c\t: choose(_,_,T), #forall(T).")
		cls.comment("1d\t: move(white,V,T), #exist(T).")
		cls.comment("1e\t: board(_,_,T), #exist(T).")
		cls.comment("1f\t: count(_,_,_,T), #exist(T).")
		cls.comment("1g\t: occupied(_,T), #exist(T),")
		cls.comment("  \t: everything else (win(_), win_row(_,_) are innermost.")

		bi := 0
		wi := 0

		for i, t := range timesZero {
			quantifier = append(quantifier, "e "+time(t))

			var s string
			var player string

			if i > 0 {
				if bi < len(blackturns) && blackturns[bi] == t {
					s = "e"
					player = black
					bi++
				}
				if wi < len(whiteturns) && whiteturns[wi] == t {
					{
						special := "a"
						for ni := 1; ni <= numbers[t]; ni++ {
							for pL := 0; pL < bitsPositions; pL++ {
								special += " " + choose(ni, pL, t)
							}
						}
						quantifier = append(quantifier, special)
					}
					s = "e"
					player = white
					wi++
				}
				for _, p := range positions {
					s += " " + move(player, p, t)
				}

				quantifier = append(quantifier, s)
			}

			{
				s = "e "
				for _, V := range positions {
					s += " " + occupied(V, t)
					for _, P := range players {
						s += " " + board(P, V, t)
					}
				}
				quantifier = append(quantifier, s)
			}

			if i > 0 {
				{
					s = "e "
					for I := 0; I <= numbers[t]+1; I++ {
						for _, V := range positionsZero {
							s += " " + count(player, V, I, t)
						}
					}
					quantifier = append(quantifier, s)
				}
			}
		}

		{
			s := "e "
			for _, P := range players {
				s += " " + win(P)
				for i, _ := range wins[P] {
					s += " " + win_row(P, i)
				}
			}
			quantifier = append(quantifier, s)
		}

		if wi != len(whiteturns) || bi != len(blackturns) {
			fmt.Println("whiteturns", whiteturns)
			fmt.Println("blackturns", blackturns)
			panic("inconsistent game description: whiteturns union blackturns = times")
		}
	}

	{
		cls.comment("1\t: ~time(T), time(T-1).")
		for _, t := range times {
			cls.add(neg(time(t)), time(pred[t]))
		}
		cls.add(time(timesZero[0]))
	}

	{
		cls.comment("2\t: ~board(P,V,T) , board(P,V,T+1).")
		for _, p := range players {
			for _, t := range times {
				for _, v := range positions {
					cls.add(neg(board(p, v, pred[t])), board(p, v, t))
				}
			}
		}
	}

	{
		cls.comment("3\t: ~board(b,V,T) , ~board(w,V,T).")
		for _, t := range times {
			for _, v := range positions {
				cls.add(neg(board(black, v, t)), neg(board(white, v, t)))
			}
		}
	}

	{
		cls.comment("4\t: occupied(V,T) <=> board(b,V,T) | board(w,V,T).")
		for _, t := range timesZero {
			for _, p := range positions {
				cls.addEquivalenceDisjunction(occupied(p, t), []string{board(black, p, t), board(white, p, t)})
			}
		}
	}

	{
		cls.comment("5\t: time(T), board(P,V,T-1) , ~board(P,V,T).")
		for _, P := range players {
			for _, T := range times {
				for _, V := range positions {
					cls.add(time(T), board(P, V, pred[T]), neg(board(P, V, T)))
				}
			}
		}
	}

	{
		cls.comment("6a\t: move(P,V,T+1), board(P,V,T) , ~board(P,V,T+1).")
		cls.comment("6c\t: board(opponent[P],V,T) , ~board(opponent[P],V,T+1). during P moves, PP won't take")
		for _, P := range players {
			for _, T := range turn[P] {
				for _, V := range positions {
					cls.add(move(P, V, T), board(P, V, pred[T]), neg(board(P, V, T)))
					cls.add(board(opponent[P], V, pred[T]), neg(board(opponent[P], V, T)))
				}
			}
		}
	}

	{
		cls.comment("8 \t: ~win(white).")
		cls.comment("9 \t: win(black).")
		cls.comment("10 \t: win(P) <=> win(P,E) : edge(P,E).")
		cls.comment("11 \t: ~win(P,E) <=> ~board(P,V,F) : in(V,E,P).")

		cls.add(win(black))
		cls.add(neg(win(white)))
		Final := times[len(times)-1]
		{
			for _, P := range players {
				edges := []string{}
				for i, ps := range wins[P] {
					row := win_row(P, i)
					edges = append(edges, row)
					ins := []string{}
					for _, V := range ps {
						ins = append(ins, neg(board(P, V, Final)))
					}
					cls.addEquivalenceDisjunction(neg(row), ins)
				}
				cls.addEquivalenceDisjunction(win(P), edges)
			}
		}

	}

	{
		cls.comment("12\t: ~count(P,V-1,I,T), count(P,V,I,T).")
		cls.comment("13\t: ~count(P,V-1,I+1,T), count(P,V,I,T).")
		cls.comment("14\t: ~move(P,V,T), ~count(P,V-1,I,T), count(P,V,I+1,T).")
		cls.comment("15\t: move(P,V,T), ~count(P,V,I,T), count(P,V-1,I,T).")
		for _, P := range players {
			for _, T := range turn[P] {
				for i, V := range positions {
					top := numbers[T] + 1
					for I := 0; I <= top; I++ {
						V1 := positionsZero[i] // V1 = V - 1
						cls.add(neg(count(P, V1, I, T)), count(P, V, I, T))
						cls.add(move(P, V, T), neg(count(P, V, I, T)), count(P, V1, I, T))
						if I != top {
							cls.add(neg(count(P, V, I+1, T)), count(P, V1, I, T))
							cls.add(neg(move(P, V, T)), neg(count(P, V1, I, T)), count(P, V, I+1, T))
						}
					}
				}
			}
		}

		cls.comment("17\t: count(P,0,0,T).")
		cls.comment("18\t: ~count(P,0,1,T).")
		cls.comment("19\t: ~count(P,|V|, N_t+1,T).")
		cls.comment("20\t: ~time(T), count(P,|V|, N_t,T).")
		for _, P := range players {
			for _, T := range turn[P] {
				V0 := positionsZero[0]
				cls.add(count(P, V0, 0, T))
				cls.add(neg(count(P, V0, 1, T)))
				cls.add(neg(time(T)), count(P, positions[len(positions)-1], numbers[T], T))
				cls.add(neg(count(P, positions[len(positions)-1], numbers[T]+1, T)))
			}
		}
	}

	{
		cls.comment("ACTIONS.")
		cls.comment("21\t: ~move(P,V,T), board(P,V,T).")
		cls.comment("22\t: ~move(P,V,T), ~occupied(V,T-1).")
		cls.comment("23\t: ~move(P,V,T), time(T).")
		for _, P := range players {
			for _, T := range turn[P] {
				for _, V := range positions {
					cls.add(neg(move(P, V, T)), board(P, V, T))
					cls.add(neg(move(P, V, T)), neg(occupied(V, pred[T])))
					cls.add(neg(move(P, V, T)), time(T))
				}
			}
		}
	}

	{

		{
			cls.comment("24\t: ~time(T), occupied(V,T-1),choose(B,I,T):V/B == 1: V, ~choose(B,T):V/B == 0: V, move(white,V,T)")
			for _, T := range turn[white] {
				for I := 1; I <= numbers[T]; I++ {
					for iV, V := range positions {
						combination := []string{}
						//						cls.comment(strconv.Itoa(iV) + " in bits " + strconv.FormatInt(int64(iV), 2))
						for i := 0; i < bitsPositions; i++ {
							if iV&(1<<uint(i)) > 0 {
								combination = append(combination, choose(I, i, T))
							} else {
								combination = append(combination, neg(choose(I, i, T)))
							}
						}
						cl := clause{neg(time(T)), occupied(V, pred[T])}
						cl = append(cl, combination...)
						cl = append(cl, move(white, V, T))
						cls.addCls(cl)
					}
				}
			}
		}

	}

	{
		cls.comment("27\t: ~occupied(V,0).")
		for _, V := range positions {
			cls.add(neg(occupied(V, timesZero[0])))
		}
	}

	if symFlag {
		cls.comment("28\t: P=black: move(black,V,1) : firstmoves(V). % break symmetries")
		T := times[0]
		if in(T, blackturns) && len(g.firstmoves) > 0 {
			cl := clause{}
			for _, V := range g.firstmoves {
				cl = append(cl, move(black, V, T))
			}
			cls.addCls(cl)
		}
	}

	{ //OUTPUT Clauses
		for _, q := range quantifier {
			fmt.Println(q)
		}

		for _, clause := range cls.cs {
			for _, c := range clause {
				fmt.Print(c, " ")
			}
			fmt.Println()
		}
	}
}

func createMap(stringSlice []string) map[string]int {
	m := make(map[string]int, len(stringSlice))
	for i, x := range stringSlice {
		m[x] = i
	}
	return m
}

func (g *game) generateBuleFacts() {

	times := g.times

	blackturns := createMap(g.blackturns)
	numbers := g.numbers

	time1 := 0
	for _, id := range times {
		player := 0
		if _, ok := blackturns[id]; ok {
			player = 1
		}
		//		fmt.Println("turnN[", i+1, ",", player, ",", numbers[id], "].")
		for i := 0; i < numbers[id]; i++ {
			time1++
			fmt.Println("turn1[", time1, ",", player, "].")
		}
	}
	fmt.Println("#const final=", time1, ".")

	positions := createMap(g.positions)
	fmt.Println("#const vertexLast=", len(g.positions)-1, ".")
	{
		player := 1
		for i, edge := range g.blackwins {
			for _, v := range edge {
				fmt.Println("edge[", player, ",", i+1, ",", positions[v], "].")
			}
		}
	}
	{
		player := 0
		for i, edge := range g.whitewins {
			for _, v := range edge {
				fmt.Println("edge[", player, ",", i+1, ",", positions[v], "].")
			}
		}
	}
}

// Maker/Maker Encoding5
// contains board and move variables
// uses counter encoding for counting moves
func (g *game) encodeMakerMaker5() {

	times := g.times
	blackturns := g.blackturns
	whiteturns := g.whiteturns
	numbers := g.numbers
	positions := g.positions

	black := "black"
	white := "white"
	players := []string{black, white}
	turn := map[string][]string{}
	turn[black] = g.blackturns
	turn[white] = g.whiteturns

	wins := map[string][][]string{}
	wins[black] = g.blackwins
	wins[white] = g.whitewins

	timesZero := append([]string{"t0_"}, times...)
	positionsZero := append([]string{"p0_"}, positions...)

	succ := map[string]string{}
	for i, t := range timesZero {
		if i == len(timesZero)-1 {
			succ[t] = False
		} else {
			succ[t] = timesZero[i+1]
		}
	}

	pred := map[string]string{}
	for i, t := range timesZero {
		if i == 0 {
			pred[t] = neg(False)
		} else {
			pred[t] = timesZero[i-1]
		}
	}

	cls = clauses{cs: []clause{}}
	quantifier := []string{}

	{ // Create quantifiers

		cls.comment("1a\t: time(T), #exist(T).")
		cls.comment("1b\t: move(black,_,T), #exist(T).")
		cls.comment("1c\t: move(white,_,T), #forall(T).")
		cls.comment("1d\t: board(_,_,T), #exist(T).")
		cls.comment("1e\t: count(_,_,_,T), #exist(T).")
		cls.comment("1e\t: occupied(_,T), #exist(T),")
		cls.comment("1f\t: stack(V,T), #exist(T).")
		cls.comment("everything else (win(_), win_row(_,_) are innermost.")

		bi := 0
		wi := 0

		for i, t := range timesZero {
			quantifier = append(quantifier, "e "+time(t))

			var s string
			var player string

			if i > 0 {
				if bi < len(blackturns) && blackturns[bi] == t {
					s = "e"
					player = black
					bi++
				}
				if wi < len(whiteturns) && whiteturns[wi] == t {
					s = "a"
					player = white
					wi++
				}
				for _, p := range positions {
					s += " " + move(player, p, t)
				}
				quantifier = append(quantifier, s)
			}

			{ // occupied and board
				s = "e "
				for _, V := range positions {
					s += " " + occupied(V, t)
					for _, P := range players {
						s += " " + board(P, V, t)
					}
				}
				quantifier = append(quantifier, s)
			}

			if i > 0 {
				{ // Count variables
					s = "e "
					for I := 0; I <= numbers[t]+1; I++ {
						for _, V := range positionsZero {
							s += " " + count(player, V, I, t)
						}
					}
					quantifier = append(quantifier, s)
				}

				if in(t, whiteturns) {
					{ /// STACK AND EXCESS(through count variables)
						s = "e"
						for _, v := range positions {
							s += " " + stack(v, t)
						}
						quantifier = append(quantifier, s)
					}
				}
			}
		}

		{
			s := "e "
			for _, P := range players {
				s += " " + win(P)
				for i, _ := range wins[P] {
					s += " " + win_row(P, i)
				}
			}
			quantifier = append(quantifier, s)
		}

		if wi != len(whiteturns) || bi != len(blackturns) {
			fmt.Println("whiteturns", whiteturns)
			fmt.Println("blackturns", blackturns)
			panic("inconsistent game description: whiteturns union blackturns = times")
		}
	}

	{
		cls.comment("1\t: ~time(T), time(T-1).")
		for _, t := range times {
			cls.add(neg(time(t)), time(pred[t]))
		}
		cls.add(time(timesZero[0]))
	}

	{
		cls.comment("2\t: ~board(P,V,T) , board(P,V,T+1).")
		for _, p := range players {
			for _, t := range times {
				for _, v := range positions {
					cls.add(neg(board(p, v, pred[t])), board(p, v, t))
				}
			}
		}
	}

	{
		cls.comment("3\t: ~board(b,V,T) , ~board(w,V,T).")
		for _, t := range times {
			for _, v := range positions {
				cls.add(neg(board(black, v, t)), neg(board(white, v, t)))
			}
		}
	}

	{
		cls.comment("4\t:~board(b,V,T), ~board(w,V,T)  <=>  ~occupied(V,T).")
		for _, t := range timesZero {
			for _, p := range positions {
				cls.addEquivalenceDisjunction(occupied(p, t), []string{board(black, p, t), board(white, p, t)})
			}
		}
	}

	{
		cls.comment("5\t: time(T), board(P,V,T-1) , ~board(P,V,T).")
		for _, P := range players {
			for _, T := range times {
				for _, V := range positions {
					cls.add(time(T), board(P, V, pred[T]), neg(board(P, V, T)))
				}
			}
		}
	}

	{
		cls.comment("8 \t: ~win(white).")
		cls.comment("9 \t: win(black).")
		cls.comment("10 \t: ~win(P,E) : edge(P,E) <=> ~win(P).")
		cls.comment("11 \t: board(P,V,Final) : in(V,E,P) <=> win(P,E).")

		cls.add(win(black))
		cls.add(neg(win(white)))
		Final := times[len(times)-1]
		{
			for _, P := range players {
				edges := []string{}
				for i, ps := range wins[P] {
					row := win_row(P, i)
					edges = append(edges, row)
					ins := []string{}
					for _, V := range ps {
						ins = append(ins, neg(board(P, V, Final)))
					}
					cls.addEquivalenceDisjunction(neg(row), ins)
				}
				cls.addEquivalenceDisjunction(win(P), edges)
			}
		}

	}

	{
		cls.comment("12\t: ~count(P,V-1,I,T), count(P,V,I,T).")
		cls.comment("13\t: ~count(P,V-1,I+1,T), count(P,V,I,T).")
		cls.comment("14\t: ~move(P,V,T), ~count(P,V-1,I,T), count(P,V,I+1,T).")
		cls.comment("15\t: move(P,V,T), count(P,V-1,I,T), ~count(P,V,I,T).")
		for _, P := range players {
			for _, T := range turn[P] {
				for i, V := range positions {
					top := numbers[T] + 1
					for I := 0; I <= top; I++ {
						V1 := positionsZero[i] // V1 = V - 1
						cls.add(neg(count(P, V1, I, T)), count(P, V, I, T))
						cls.add(move(P, V1, T), count(P, V1, I, T), neg(count(P, V, I, T)))
						if I != top {
							cls.add(neg(count(P, V, I+1, T)), count(P, V1, I, T))
							cls.add(neg(move(P, V, T)), neg(count(P, V1, I, T)), count(P, V, I+1, T))
						}
					}
				}
			}
		}

		cls.comment("17\t: count(P,0,0,T).")
		cls.comment("18\t: ~count(P,0,1,T).")
		for _, P := range players {
			for _, T := range turn[P] {
				V0 := positionsZero[0]
				cls.add(count(P, V0, 0, T))
				cls.add(neg(count(P, V0, 1, T)))
			}
			if P == black {

			}

		}
	}

	{
		cls.comment("19\t: ~time(T), count(black,|V|, N_t,T).")
		cls.comment("20\t: ~count(black,|V|, N_t+1,T).")
		cls.comment("21\t: ~move(black,V,T), time(T).")
		cls.comment("22\t: ~move(black,V,T), ~occupied(V,T-1).")
		cls.comment("23\t: ~move(black,V,T), board(black,V,T).")
		for _, T := range turn[black] {
			cls.add(neg(time(T)), count(black, positions[len(positions)-1], numbers[T], T))
			cls.add(neg(count(black, positions[len(positions)-1], numbers[T]+1, T)))
			for _, V := range positions {
				cls.add(neg(move(black, V, T)), time(T))
				cls.add(neg(move(black, V, T)), neg(occupied(V, pred[T])))
				cls.add(neg(move(black, V, T)), board(black, V, T))
			}
		}

		{
			cls.comment("6a\t: move(black,V,T+1), board(black,V,T) , ~board(black,V,T+1).")
			cls.comment("6c\t: T is black move: board(white,V,T) , ~board(white,V,T+1). during black moves, white won't take")
			P := black
			PP := white
			for _, T := range turn[P] {
				for _, V := range positions {
					cls.add(move(P, V, T), board(P, V, pred[T]), neg(board(P, V, T)))
					cls.add(board(PP, V, pred[T]), neg(board(PP, V, T)))
				}
			}
		}
	}

	cls.comment("6c\t: board(P,V,T), ~board(P,V,T+1) <=> move(P,V,T+1).")
	cls.comment("6d\t: T is PP move: board(P,V,T) , ~board(P,V,T+1).")
	cls.comment("7d\t: ~stack(white,V,T), move(white,V,T).")
	cls.comment("7d\t: ~stack(white,V,T), occupied(V,T-1).")
	for _, P := range players {
		for _, T := range turn[P] {
			for _, V := range positions {
				cls.addEquivalenceDisjunction(move(P, V, T), []string{board(P, V, pred[T]), neg(board(P, V, T))})
			}
		}
	}

	{
		{
			cls.comment("24\t: ~move(white,V,T+1), ~board(white,V,T) , ~board(white,V,T+1)..")
			cls.comment("25\t: ~move(white,V,T+1), board(white,V,T) , ~board(white,V,T+1).")
			cls.comment("25\t: X=white: ~step(X,V,I,T) <=> ~count(X,V,I,T) | count(X,V-1,I,T).")
			X := white
			for _, T := range turn[X] {
				for i, V := range positions {
					nums := make([]string, 0)
					for I := 1; I <= numbers[T]; I++ {
						V1 := positionsZero[i] // V1 = V - 1
						cls.addEquivalenceDisjunction(neg(step(X, V, I, T)), []string{neg(count(X, V, I, T)), count(X, V1, I, T)})
						if I > 0 {
							nums = append(nums, step(X, V, I, T))
						}
					}
					cls.addEquivalenceDisjunction(emove(X, V, T), nums)
				}
			}
		}

		{
			cls.comment("26b\t: ~time(T), occupied(V,T-1), ~emove(white,V,T), board(white,V,T).")
			for _, T := range turn[white] {

				for _, V := range positions {
					cl := clause{neg(time(T)), occupied(V, pred[T]), neg(emove(white, V, T)), board(white, V, T)}
					//for _, pp := range connections { TODO FIX CONNECTION
					//	if pp[1] == V {
					//		cl = append(cl, occupied(pp[0], pred[T]))
					//	}
					//}
					cls.addCls(cl)
				}
			}
		}

		{
			cls.comment("6b\t: P=white: emove(P,V,T), board(P,V,T-1) , ~board(P,V,T).")
			cls.comment("6d\t: PP=black: board(PP,V,T) , ~board(PP,V,T+1). during white moves, black won't take")
			P := white
			PP := black
			for _, T := range turn[P] {
				for _, V := range positions {
					cls.add(emove(P, V, T), board(P, V, pred[T]), neg(board(P, V, T)))
					cls.add(board(PP, V, pred[T]), neg(board(PP, V, T)))
				}
			}
		}

	}

	{
		cls.comment("27\t: ~occupied(V,0).")
		for _, V := range positions {
			cls.add(neg(occupied(V, timesZero[0])))
		}
	}

	if symFlag {
		cls.comment("28\t: P=black: move(black,V,1) : firstmoves(V). % break symmetries")
		T := times[0]
		if in(T, blackturns) && len(g.firstmoves) > 0 {
			cl := clause{}
			for _, V := range g.firstmoves {
				cl = append(cl, move(black, V, T))
			}
			cls.addCls(cl)
		}
	}

	{ //OUTPUT Clauses
		for _, q := range quantifier {
			fmt.Println(q)
		}

		for _, clause := range cls.cs {
			for _, c := range clause {
				fmt.Print(c, " ")
			}
			fmt.Println()
		}
	}
}

// Maker/Breaker Encoding
// whitewins is empty
// black starts, alternating moves, single moves per position
// no symmetry breaking
// (encoding for HEX, and others such games)
// WORK IN PROGESS
func (g *game) encodeMakerBreaker() {

	times := g.times
	blackturns := g.blackturns
	whiteturns := g.whiteturns
	numbers := g.numbers
	positions := g.positions
	blackwins := g.blackwins

	cheat := "cheat"
	win := "win"
	black := "B"
	white := "W"

	cls = clauses{cs: []clause{}}
	quantifier := []string{}

	{
		// check prerequisites
		isGoodForMBSpecial := times[len(times)-1] != blackturns[len(blackturns)-1]
		for i, a := range times {
			if numbers[a] != 1 {
				isGoodForMBSpecial = false
			}
			if i%2 == 0 {
				isGoodForMBSpecial = a == blackturns[i/2]
			}
			if i%2 == 1 {
				isGoodForMBSpecial = a == whiteturns[i/2]
			}

			if !isGoodForMBSpecial {
				fmt.Println("ERROR: encoding not suited for MB ")
				os.Exit(1)
			}
		}
	}

	quantifierAux := make(map[string]int, 0)
	quantifierDepth := 0
	{ // Create quantifiers
		for _, t := range times {
			var s string
			if in(t, blackturns) {
				quantifierAux[t] = quantifierDepth
				s = "e"
				for _, p := range positions {
					s += " " + board(black, p, t)
				}
				if moveFlag {
					for _, p := range positions {
						s += " " + move(black, p, t)
					}
				}
				quantifier = append(quantifier, s)
				quantifierDepth++
			}

			if in(t, whiteturns) {
				s = "a"
				for _, p := range positions {
					s += " " + move(white, p, t)
				}
				quantifier = append(quantifier, s)
				quantifierDepth++

				s = "e"
				if cntFlag && len(positions) >= cntThresh {
					// will be added later through counter encoding
				} else {
					subsets := allSubsetsSize(positions, numbers[t]+1)
					for _, subset := range subsets {
						s += " " + excess(subset, t)
					}
				}
				if stackFlag {
					for _, p := range positions {
						s += " " + stack(p, t)
					}
				}

				quantifier = append(quantifier, s)
				quantifierAux[t] = quantifierDepth
				quantifierDepth++
			}
		}

		{
			s := "e " + win
			for i, _ := range blackwins {
				s += " " + win_row_s(i)
			}
			quantifier = append(quantifier, s)
		}
	}

	{
		cls.comment("c26")
		cls.comment("succ(I,J), board(black,P,I) -> board(black,P,J)")
		for i := 0; i < len(blackturns)-1; i++ {
			for _, p := range positions {
				cls.add(neg(board(black, p, blackturns[i])), board(black, p, blackturns[i+1]))
			}
		}
	}

	if moveFlag {
		cls.comment("move(B,P,J) <=> ~board(B,P,I), board(B,P,J), succ(I,J).")
		for i := 0; i < len(blackturns)-1; i++ {
			for _, p := range positions {
				cls.add(move(black, p, blackturns[i+1]), board(black, p, blackturns[i]), neg(board(black, p, blackturns[i+1])))
				cls.add(move(black, p, blackturns[0]), neg(board(black, p, blackturns[0])))
				cls.add(neg(move(black, p, blackturns[i])), board(black, p, blackturns[i]))
				cls.add(neg(move(black, p, blackturns[i+1])), neg(board(black, p, blackturns[i])))
			}
		}
	}

	var cheats []string
	{
		cls.comment("c14")
		cls.comment("subset(Element,Subset,Size,Set),excess(Subset,T) -> move(white,P,T)")
		if cntFlag && len(positions) >= cntThresh {
			for _, t := range whiteturns {
				moves := []string{}
				for _, p := range positions {
					moves = append(moves, move(white, p, t))
				}
				_, l2, aux := atLeastTwo(moves, "atLeast2("+t+")")
				for _, a := range aux {
					quantifier[quantifierAux[t]] += " " + a
				}
				cheats = append(cheats, l2)
			}
		} else {
			for _, t := range whiteturns {
				subsets := allSubsetsSize(positions, numbers[t]+1)
				for _, subset := range subsets {
					for _, p := range subset {
						cls.add(neg(excess(subset, t)), move(white, p, t))
					}
					cheats = append(cheats, excess(subset, t))
				}
			}
		}
	}

	if stackFlag {
		cls.comment("stack")
		// assumes correct order
		for i, t := range whiteturns {
			for _, p := range positions {
				cls.add(neg(stack(p, t)), board(black, p, blackturns[i]))
				cls.add(neg(stack(p, t)), move(white, p, t))
				cheats = append(cheats, stack(p, t))
			}

		}
	}

	{
		cls.comment("c27")
		subsets := allSubsetsSize(positions, numbers[blackturns[0]]+1)
		for _, subset := range subsets {
			c := clause{}
			for _, p := range subset {
				c = append(c, neg(board(black, p, blackturns[0])))
			}
			cls.addCls(c)
		}

		for i := 0; i < len(blackturns)-1; i++ {
			subsets := allSubsetsSize(positions, numbers[blackturns[i]]+1)
			for _, subset := range subsets {
				c := clause{}
				for _, p := range subset {
					c = append(c, board(black, p, blackturns[i]), neg(board(black, p, blackturns[i+1])))
				}
				cls.addCls(c)
			}
		}
	}

	{
		cls.comment("c28")
		for i := 0; i < len(blackturns)-1; i++ {
			for _, p := range positions {
				cls.add(board(black, p, blackturns[i]), neg(move(white, p, whiteturns[i])), neg(board(black, p, blackturns[len(blackturns)-1])))
			}
		}
	}

	{
		cls.comment("c29")
		cls.add(cheat, win)

		b_win := clause{neg(win)}
		for i, ps := range blackwins {
			row := win_row_s(i)
			b_win = append(b_win, row)
			winImpl := clause{row}
			for _, p := range ps {
				cls.add(neg(row), board(black, p, blackturns[len(blackturns)-1])) // (10)
				winImpl = append(winImpl, neg(final(black, p)))
			}
			if c10Flag {
				cls.comment("c10")
				cls.addCls(winImpl)
			}
		}
		cls.addCls(b_win) // (9)

		cl := clause{neg(cheat)}
		for _, x := range cheats {
			cl = append(cl, x)
		}
		cls.addCls(cl)
	}

	{ //OUTPUT Clauses
		for _, q := range quantifier {
			fmt.Println(q)
		}

		for _, clause := range cls.cs {
			for _, c := range clause {
				fmt.Print(c, " ")
			}
			fmt.Println()
		}
	}
}

// ExactlyOneSpecial treats the case of order encoding.
// Essentially it encodes |lits1|+1 = |lits2|
func exactlyOneSpecial(lits1 []string, lits2 []string, id string) (string, string) {
	nlits := len(lits1)

	if nlits != len(lits2) {
		fmt.Println(lits1, lits2, id)
		panic("exactlyOneSpecial called with wrong input")
	}

	for i := 0; i < nlits; i++ {
		for j := 0; j <= 2; j++ {
			cls.add(neg(cnt(i-1, j, id)), cnt(i, j, id))

			cls.add(neg(cnt(i-1, j-1, id)), cnt(i, j, id), lits1[i], neg(lits2[i]))

			cls.add(cnt(i-1, j, id), neg(cnt(i, j, id)), neg(lits1[i]))
			cls.add(cnt(i-1, j, id), neg(cnt(i, j, id)), lits2[i])

			cls.add(cnt(i-1, j-1, id), neg(cnt(i, j, id)))
		}
	}
	return cnt(nlits-1, 1, id), cnt(nlits-1, 2, id)
}

func exactlyOne(lits []string, id string) (string, string) {

	for i, l := range lits {
		for j := 0; j <= 2; j++ {

			cls.add(neg(cnt(i-1, j, id)), cnt(i, j, id))
			cls.add(neg(cnt(i-1, j-1, id)), cnt(i, j, id), neg(l))

			cls.add(cnt(i-1, j, id), neg(cnt(i, j, id)), l)
			cls.add(cnt(i-1, j-1, id), neg(cnt(i, j, id)))
		}
	}

	return cnt(len(lits)-1, 1, id), cnt(len(lits)-1, 2, id)
}

func atLeastTwo(lits []string, id string) (atLeast1 string, atLeast2 string, aux []string) {

	for i, l := range lits {
		for j := 0; j <= 2; j++ {

			//cls.add(neg(cnt(i-1, j, id)), cnt(i, j, id))
			//cls.add(neg(cnt(i-1, j-1, id)), cnt(i, j, id), neg(lits[i]))

			cls.add(cnt(i-1, j, id), neg(cnt(i, j, id)), l)
			cls.add(cnt(i-1, j-1, id), neg(cnt(i, j, id)))

			if cnt(i, j, id) != False && cnt(i, j, id) != neg(False) {
				aux = append(aux, cnt(i, j, id))
			}
		}
	}
	return cnt(len(lits)-1, 1, id), cnt(len(lits)-1, 2, id), aux
}

func count(P string, V string, I int, T string) string {
	return fmt.Sprintf("count(%v,%v,%v,%v)", P, V, I, T)
}

func cnt(i int, j int, l string) string {
	if j <= 0 {
		return neg(False)
	}
	if j-i >= 2 {
		return False
	}
	return fmt.Sprintf("cnt(%v,%v)_%v", i, j, l)
}

func win_row_s(row int) string {
	return "win(" + strconv.Itoa(row) + ")"
}

func win_row(P string, row int) string {
	return "win(" + P + ")_" + strconv.Itoa(row)
}

func win(P string) string {
	return "win(" + P + ")"
}

func time(T string) string {
	return "time(" + T + ")"
}

func stack(p string, T string) string {
	return "stack(" + p + "," + T + ")"
}

func excess(set []string, T string) string {
	s := ""
	for _, x := range set {
		s += x
	}
	return "excess(" + s + "," + T + ")"
}

func board(P, V, T string) string {
	return "board(" + P + "," + V + "," + T + ")"
}

func occupied(V, T string) string {
	return "occupied(" + V + "," + T + ")"
}

func move(P, V, T string) string {
	return "move(" + P + "," + V + "," + T + ")"
}

func emove(P, V, T string) string {
	return "emove(" + P + "," + V + "," + T + ")"
}

func choose(Ni int, B int, T string) string {
	return "choose(" + strconv.Itoa(Ni) + "," + strconv.Itoa(B) + "," + T + ")"
}

func step(P string, V string, J int, T string) string {
	return "step(" + P + "," + V + "," + strconv.Itoa(J) + "," + T + ")"
}

func final(P, V string) string {
	return "final(" + P + "," + V + ")"
}

func neg(s string) string {
	if strings.HasPrefix(s, "~") {
		return strings.TrimLeft(s, "~")
	}
	return "~" + s
}

func parse() (game, error) {

	var g game

	if len(flag.Args()) == 0 {
		return g, errors.New("usage: ./ground <filename>")
	}
	file, err := os.Open(flag.Args()[0])
	if err != nil {
		return g, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	var state string

	for scanner.Scan() {

		s := scanner.Text()
		if pos := strings.Index(s, "%"); pos >= 0 {
			s = s[:pos]
		}
		fields := strings.Fields(s)

		if len(fields) == 0 || strings.HasPrefix(fields[0], "%") {
			continue
		}

		first := strings.ToLower(fields[0])
		if strings.HasPrefix(first, "#") {
			switch first {
			case "#positions":
				state = first
				continue
			case "#blackwins":
				state = first
				continue
			case "#whitewins":
				state = first
				continue
			case "#times":
				state = first
				continue
			case "#blackturns":
				state = first
				continue
			case "#blackinitials":
				state = first
				continue
			case "#whiteinitials":
				state = first
				continue
			case "#firstmoves":
				state = first
				continue
			default:
				return g, errors.New("code word unknown:" + first)
			}
		}

		switch state {
		case "#positions":
			g.positions = append(g.positions, fields...)
		case "#blackwins": // slice of slices
			g.blackwins = append(g.blackwins, fields)
		case "#whitewins": // slice of slice
			g.whitewins = append(g.whitewins, fields)
		case "#times":
			g.times = append(g.times, fields...)
		case "#blackturns":
			g.blackturns = append(g.blackturns, fields...)
		case "#blackinitials":
			g.blackinitials = append(g.blackinitials, fields...)
		case "#whiteinitials":
			g.whiteinitials = append(g.whiteinitials, fields...)
		case "#firstmoves":
			g.firstmoves = append(g.firstmoves, fields...)
		}

		// swap white and black player
		if swpFlag {
			g.blackinitials, g.whiteinitials = g.whiteinitials, g.blackinitials
			g.blackwins, g.whitewins = g.whitewins, g.blackwins
			g.blackturns = remove(g.times, g.blackturns)
		}

	}

	{ // CHECK CONSISTENCY
		if len(g.positions) == 0 {
			return g, errors.New("positions empty")
		}
		for _, x := range g.blackinitials {
			if !in(x, g.positions) {
				fmt.Println(x, "is not a position", g.blackinitials)
			}
		}
		for _, x := range g.whiteinitials {
			if !in(x, g.positions) {
				fmt.Println(x, "is not a position", g.whiteinitials)
			}
		}
		for _, x := range g.firstmoves {
			if !in(x, g.positions) {
				fmt.Println(x, "is not a position", g.firstmoves)
			}
		}
	}

	{ // Cleanup remove initials from positions
		g.positions = remove(g.positions, g.blackinitials)
		g.positions = remove(g.positions, g.whiteinitials)
	}

	if len(g.positions) < len(g.times) {
		//  remove times if not enough positions.
		removeTimes := g.times[len(g.positions):]
		g.times = g.times[:len(g.positions)]
		g.positions = remove(g.positions, removeTimes)
		g.positions = remove(g.positions, removeTimes)
	}

	{ // remove winning positions contains initial from opponent
		for _, w := range g.whiteinitials {
			tmp := g.blackwins[:0]
			for _, bs := range g.blackwins {
				if !in(w, bs) {
					bs = remove(bs, g.blackinitials)
					tmp = append(tmp, bs)
				}
			}
			g.blackwins = tmp
		}
		for _, b := range g.blackinitials {
			tmp := g.whitewins[:0]
			for _, ws := range g.whitewins {
				if !in(b, ws) {
					ws = remove(ws, g.whiteinitials)
					tmp = append(tmp, ws)
				}
			}
			g.whitewins = tmp
		}
	}

	for i, j := 0, 0; i < len(g.times); i++ {
		if j < len(g.blackturns) && g.times[i] == g.blackturns[j] {
			j++
		} else {
			g.whiteturns = append(g.whiteturns, g.times[i])
		}
	}

	return g, nil
}

// keep order in slice
func in(e string, set []string) bool {
	for _, x := range set {
		if e == x {
			return true
		}
	}
	return false
}

// generates a new slice that contains all elements of ms that do not occur in rm
func remove(ms, rm []string) (ms2 []string) {
	position_set := make(map[string]bool, len(ms))
	for _, x := range rm {
		position_set[x] = true
	}
	for _, x := range ms {
		if !position_set[x] {
			ms2 = append(ms2, x)
		}
	}
	return
}

func allSubsetsSize(input []string, n int) (subsets [][]string) {
	tmp := []string{}
	acc := make([][]string, 0, len(input)*len(input)/2)
	subset(input, n, 0, tmp, &acc)
	return acc
}

func subset(arr []string, left int, index int, l []string, acc *[][]string) {
	if left == 0 {
		nl := make([]string, len(l))
		copy(nl, l)
		*acc = append(*acc, nl)
		return
	}

	for i := index; i < len(arr); i++ {
		l = append(l, arr[i])
		subset(arr, left-1, i+1, l, acc)
		l = l[:len(l)-1]
	}
}
