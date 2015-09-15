package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/gonum/matrix/mat64"
)

var graph = flag.Bool("graph", false, "print the graphviz dot script of match relationships")

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

func (t Tournament) Players() map[int]string {
	ids := t.Ids()
	players := map[int]string{}
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

	for i := 0; i < n; i++ {
		for j := i; j < n; j++ {
			if m.At(i, j) > 0 || m.At(j, i) > 0 {
				m.Set(i, j, m.At(i, j)/(m.At(j, i)+m.At(i, j)))
			}
		}
	}
	for i := 0; i < n; i++ {
		for j := 0; j < i; j++ {
			if m.At(j, i) > 0 {
				m.Set(i, j, 1-m.At(j, i))
			}
		}
	}
	return m
}

func main() {
	flag.Parse()

	tourn := Tournament{
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

	m := tourn.Matrix()

	if *graph {
		Graph(os.Stdout, m, tourn.Players())
		return
	}

	prefix := "tournmat = "
	fmt.Printf("%v%v\n", prefix, mat64.Formatted(m, mat64.Prefix(strings.Repeat(" ", len(prefix)))))

	fmt.Println()

	eig := mat64.Eigen(m, 1e-10)
	prefix = "eigvects = "
	fmt.Printf("%v%.4v\n", prefix, mat64.Formatted(eig.V, mat64.Prefix(strings.Repeat(" ", len(prefix)))))

}

func Graph(w io.Writer, m *mat64.Dense, players map[int]string) {
	fmt.Fprintf(w, "digraph matches {\n")
	r, c := m.Dims()
	for i := 0; i < r; i++ {
		for j := 0; j < c; j++ {
			if v := m.At(i, j); v > 0 {
				winner, ok := players[i]
				if !ok {
					winner = strconv.Itoa(i)
				}
				loser, ok := players[i]
				if !ok {
					loser = strconv.Itoa(i)
				}

				fmt.Fprintf(w, "\"%v\" -> \"%v\";\n", winner, loser)
			}
		}
	}
	fmt.Fprintf(w, "}\n")
}
