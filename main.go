package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"strings"

	"github.com/gonum/matrix/mat64"
)

var graph = flag.Bool("graph", false, "print the graphviz dot script of match relationships")
var matrix = flag.Bool("matrix", false, "print the tournament matrix")
var eigvect = flag.Bool("eigvect", false, "print the tournament eigenvectors")
var eigval = flag.Bool("eigval", false, "print the tournament eigenvalues")
var demo = flag.Bool("demo", false, "just use demo tournament matches")

func main() {
	flag.Parse()
	log.SetFlags(0)

	var tourn Tournament

	if *demo {
		tourn = Tournament{
			{"bob-r", "joe-c"},
			{"bob-r", "tim-c"},
			{"bob-r", "tim-c"},
			{"bob-r", "tim-c"},
			{"joe-r", "tim-c"},
			{"joe-r", "bob-c"},
			{"tim-r", "joe-c"},
			{"tim-r", "bob-c"},
			{"bob-c", "joe-r"},
			{"bob-c", "tim-r"},
			{"joe-c", "tim-r"},
			{"joe-c", "bob-r"},
			{"tim-c", "joe-r"},
			{"tim-c", "bob-r"},
		}
	} else {
		matches, err := ParseMatches(os.Stdin)
		if err != nil {
			log.Fatal(err)
		}
		tourn = Tournament(matches)
	}

	if *graph {
		tourn.Graph(os.Stdout)
		return
	} else if *matrix {
		prefix := "tournmat = "
		fmt.Printf("%v%v\n", prefix, mat64.Formatted(tourn.Matrix(), mat64.Prefix(strings.Repeat(" ", len(prefix)))))
		return
	} else if *eigvect {
		eig := mat64.Eigen(tourn.Matrix(), 1e-10)
		prefix := "eigvects = "
		fmt.Printf("%v%.4v\n", prefix, mat64.Formatted(eig.V, mat64.Prefix(strings.Repeat(" ", len(prefix)))))
		return
	} else if *eigval {
		eig := mat64.Eigen(tourn.Matrix(), 1e-10)
		prefix := "eigvals = "
		fmt.Printf("%v%.4v\n", prefix, mat64.Formatted(eig.D(), mat64.Prefix(strings.Repeat(" ", len(prefix)))))
		return
	}

	ranks := tourn.Ranks()
	if ranks == nil {
		log.Fatalf("no valid eigenvector ranking found")
	}

	players := tourn.Players()
	for i, rank := range ranks {
		fmt.Printf("%v\t%.2v\n", players[i], rank)
	}

}

func ParseMatches(r io.Reader) ([]Match, error) {
	s := bufio.NewScanner(os.Stdin)
	s.Split(func(data []byte, atEOF bool) (advance int, token []byte, err error) {
		i := bytes.IndexAny(data, "\n \t\r")
		if len(data) == 0 {
			return 0, nil, nil
		} else if i == -1 {
			return len(data), data, nil
		}

		token = bytes.TrimSpace(data[:i])
		if len(token) == 0 {
			token = nil
		}
		return i + 1, token, nil
	})

	matches := []Match{}
	for {
		if !s.Scan() {
			break
		}
		winner := s.Text()
		if !s.Scan() {
			return nil, errors.New("odd number of players causes opponentless match")
		}
		loser := s.Text()

		matches = append(matches, Match{winner, loser})
	}
	return matches, s.Err()
}

type Match struct {
	Winner string
	Loser  string
}

type Tournament []Match

func (t Tournament) Ids() map[string]int {
	ids := map[string]int{}
	id := 0
	for _, match := range t {
		if _, ok := ids[match.Winner]; !ok {
			ids[match.Winner] = id
			id++
		}
		if _, ok := ids[match.Loser]; !ok {
			ids[match.Loser] = id
			id++
		}
	}
	return ids
}

func (t Tournament) Players() []string {
	ids := t.Ids()
	players := make([]string, len(ids))
	for player, id := range ids {
		players[id] = player
	}
	return players
}

func (t Tournament) Matrix() *mat64.Dense {
	players := t.Ids()
	n := len(players)
	m := mat64.NewDense(n, n, nil)

	for _, match := range t {
		r := players[match.Winner]
		c := players[match.Loser]
		m.Set(r, c, m.At(r, c)+1)
	}
	return m
}

func (t Tournament) Ranks() []float64 {
	m := t.Matrix()
	eig := mat64.Eigen(m, 1e-10)
	_, c := eig.V.Dims()
	var ranks []float64
	for i := 0; i < c; i++ {
		ranks = eig.V.Col(nil, i)
		sense := math.Copysign(1, ranks[0])
		for _, val := range ranks {
			if sense == 0 {
				sense = math.Copysign(1, val)
			}
			if val*sense < 0 {
				ranks = nil
				break
			}
		}

		if ranks != nil {
			min := 1e100
			max := 0.0
			for i := range ranks {
				col := t.Matrix().Col(nil, i)
				row := t.Matrix().Row(nil, i)
				tot := 0.0
				for j := range col {
					tot += col[j] + row[j]
				}
				ranks[i] = ranks[i] * sense / tot // normalize to # games
				min = math.Min(ranks[i], min)
				max = math.Max(ranks[i], max)
			}

			// uncomment this to make min score = 0 and max score = 1
			//for i := range ranks {
			//	ranks[i] = (ranks[i] - min) / (max - min)
			//}
			break
		}
	}

	return ranks
}

func (t Tournament) Graph(w io.Writer) {
	players := t.Players()
	m := t.Matrix()
	ranks := t.Ranks()

	fmt.Fprintf(w, "digraph matches {\n")

	min := ranks[0]
	max := ranks[0]
	for _, v := range ranks[1:] {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	bound := max - min

	for i, player := range players {
		width := .5 + 2*(ranks[i]-min)/bound
		height := .5 + 1.3*(ranks[i]-min)/bound
		red := uint8((ranks[i] - min) / bound * 255)
		green := 255 - red
		blue := green
		fmt.Fprintf(w, "\"%v\" [width=%.2f, height=%.2f, label=\"%v\", style=filled, fillcolor=\"#FF%02x%02x\"];\n", player, width, height, fmt.Sprintf("%v\\n(%.2v)", player, ranks[i]), green, blue)
	}

	r, c := m.Dims()
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			if v := m.At(i, j); v > 0 {
				winner := players[i]
				loser := players[j]

				for n := 0; n < int(v); n++ {
					fmt.Fprintf(w, "\"%v\" -> \"%v\";\n", loser, winner)
				}
			}
		}
	}
	fmt.Fprintf(w, "}\n")
}
