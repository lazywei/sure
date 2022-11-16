package main

import (
	"flag"
	"fmt"
	"io/fs"
	"math"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var pattern = regexp.MustCompile(`\[([^\[]*)\]\(([^\(]*\.md)\)`)

type Graph struct {
	nodes map[string]*Node
}

type Node struct {
	path     string
	inLinks  []*Link
	outLinks []*Link
}

type Link struct {
	row     int
	col     int
	from    *Node
	to      *Node
	context []string
}

func NewNode(path string) *Node {
	return &Node{
		path:     path,
		inLinks:  []*Link{},
		outLinks: []*Link{},
	}
}

func (n *Node) LinkTo(another *Node, row, col int, context []string) {
	link := &Link{row: row, col: col, from: n, to: another, context: context}
	n.outLinks = append(n.outLinks, link)
	another.inLinks = append(another.inLinks, link)
}

func (n *Node) DescribeInbounds() {
	fmt.Printf("%s is mentioned in\n", n.path)
	for _, inLink := range n.inLinks {
		fmt.Printf("  %s\n", inLink.from.path)
		for i, ctxLine := range inLink.context {
			fmt.Printf("    %d: %s\n", inLink.row+i, ctxLine)
		}
	}
}

func BuildGraph(rootDir string) (*Graph, int, error) {
	graph := &Graph{
		nodes: map[string]*Node{},
	}
	nMDs := 0
	err := filepath.WalkDir(rootDir, func(curPath string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(curPath, ".md") {
			return nil
		}

		nMDs += 1
		content, err := os.ReadFile(curPath)
		if err != nil {
			return err
		}
		curPathRel, err := filepath.Rel(rootDir, curPath)
		if err != nil {
			return err
		}
		node := graph.GetNode(curPathRel)
		lines := strings.Split(string(content), "\n")
		nLines := len(lines)
		for row := 0; row < len(lines); row++ {
			line := lines[row]
			for _, locs := range pattern.FindAllStringSubmatchIndex(line, -1) {
				if len(locs) != 6 {
					panic(fmt.Sprintf("locs size != 6, line = %s, locs = %v", line, locs))
				}
				mentionAt := locs[2]
				pathOfLink, err := url.QueryUnescape(line[locs[4]:locs[5]])
				if err != nil {
					return err
				}
				// normalize the path so every path is relative to the project root
				// (e.g. ../foo.md to ./foo.md)
				pathOfLink = filepath.Join(filepath.Dir(curPath), "./", pathOfLink)
				pathOfLinkRel, err := filepath.Rel(rootDir, pathOfLink)
				if err != nil {
					return err
				}
				nodeOfLink := graph.GetNode(pathOfLinkRel)
				contextStart, contextEnd := int(math.Max(0, float64(row-1))), int(math.Min(float64(nLines), float64(row+3)))
				context := lines[contextStart:contextEnd]
				node.LinkTo(nodeOfLink, row, mentionAt, context)
			}
		}

		return nil
	})
	return graph, nMDs, err
}

func (g *Graph) GetNode(path string) *Node {
	if node, ok := g.nodes[path]; ok {
		return node
	}
	node := NewNode(path)
	g.nodes[path] = node
	return node
}

func (g *Graph) DescribeAllNodes() {
	for path, node := range g.nodes {
		if path != node.path {
			panic(fmt.Sprintf("path != node.path (%s != %s)", path, node.path))
		}
		if len(node.inLinks) > 0 {
			node.DescribeInbounds()
		}
	}
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	rootDir := flag.String("root", cwd, "root directory")
	linkToFlag := flag.String("link-to", "", "filename linked to")
	flag.Parse()

	// if len(*linkToFlag) == 0 {
	// 	fmt.Println("link-to must not be empty")
	// 	flag.PrintDefaults()
	// 	os.Exit(1)
	// }

	graph, cnt, err := BuildGraph(*rootDir)
	if err != nil {
		fmt.Printf("Error while walking, err: %v\n", err)
		os.Exit(1)
	}
	if cnt != len(graph.nodes) {
		fmt.Printf("Visited files mismatch: %d != %d (# of graph nodes)\n", cnt, len(graph.nodes))
		os.Exit(1)
	}

	if len(*linkToFlag) == 0 {
		graph.DescribeAllNodes()
		return
	}

	normalizedLinkTo, err := filepath.Rel(*rootDir, filepath.Join(cwd, "./", *linkToFlag))
	if err != nil {
		fmt.Printf("Failed to normalize link to: %s, err: %v\n", *linkToFlag, err)
		os.Exit(1)
	}

	node, ok := graph.nodes[normalizedLinkTo]
	if !ok {
		fmt.Printf("No file found: %s\n", normalizedLinkTo)
		os.Exit(1)
	}
	node.DescribeInbounds()
}
