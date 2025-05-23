package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"os"
	"sort"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type Combination struct {
	Root  string `json:"root"`
	Left  string `json:"left"`
	Right string `json:"right"`
	Tier  int    `json:"tier,string"`
}

type Node struct {
	Element string
	Left    *Node
	Right   *Node
}

var combinations map[string][]Combination
var tierMap map[string]int
var BFSVisitedCount int
var MultiVisitedCount int32
var DFSVisitedCount int
var BidirectionalVisitedCount int
var reverseMap map[string][]string

func LoadCombinations(filename string) error {
	data, err := os.ReadFile(filename)
	if err != nil {
		return err
	}

	var raw []Combination

	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	combinations = make(map[string][]Combination)
	tierMap = make(map[string]int)
	reverseMap = make(map[string][]string)

	for _, c := range raw {
		combinations[c.Root] = append(combinations[c.Root], c)
		tierMap[c.Root] = c.Tier

		reverseMap[c.Left] = append(reverseMap[c.Left], c.Root)
		reverseMap[c.Right] = append(reverseMap[c.Right], c.Root)
	}
	return nil
}

func isBasic(element string) bool {
	basics := map[string]bool{
		"Earth": true,
		"Air":   true,
		"Water": true,
		"Fire":  true,
		"Time":  true,
	}
	return basics[element]
}

type RecipePath struct {
	Left  string
	Right string
}

func FindRecipeBFS(target string) *Node {
	fmt.Printf("\n=== Starting BFS search for: %s ===\n", target)
	
	if isBasic(target) {
		fmt.Printf("Found basic element: %s\n", target)
		BFSVisitedCount = 1
		return &Node{Element: target}
	}

	if _, exists := combinations[target]; !exists {
		fmt.Printf("Element %s not found in combinations\n", target)
		BFSVisitedCount = 0
		return nil
	}

	fmt.Printf("Found %d combinations for %s\n", len(combinations[target]), target)
	visited := make(map[string]bool)
	recipeMap := make(map[string]*Node)
	queue := []string{target}
	BFSVisitedCount = 0

	fmt.Println("\nFirst pass: Collecting combinations...")
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true
		BFSVisitedCount++
		fmt.Printf("Visiting: %s (visited count: %d)\n", current, BFSVisitedCount)

		if isBasic(current) {
			fmt.Printf("Found basic element: %s\n", current)
			recipeMap[current] = &Node{Element: current}
			continue
		}

		validCombos := []Combination{}
		for _, comb := range combinations[current] {
			if tierMap[comb.Left] < tierMap[current] && tierMap[comb.Right] < tierMap[current] {
				validCombos = append(validCombos, comb)
			}
		}

		sort.SliceStable(validCombos, func(i, j int) bool {
			iDiff := (tierMap[current] - tierMap[validCombos[i].Left]) + (tierMap[current] - tierMap[validCombos[i].Right])
			jDiff := (tierMap[current] - tierMap[validCombos[j].Left]) + (tierMap[current] - tierMap[validCombos[j].Right])
			return iDiff > jDiff
		})

		for _, comb := range validCombos {
			fmt.Printf("  Checking combination: %s (tier %d) + %s (tier %d) = %s (tier %d)\n", 
				comb.Left, tierMap[comb.Left], comb.Right, tierMap[comb.Right], comb.Root, comb.Tier)
			
			if !visited[comb.Left] {
				queue = append(queue, comb.Left)
				fmt.Printf("    Added to queue: %s\n", comb.Left)
			}
			if !visited[comb.Right] {
				queue = append(queue, comb.Right)
				fmt.Printf("    Added to queue: %s\n", comb.Right)
			}
		}
	}

	fmt.Println("\nSecond pass: Building recipes...")
	changed := true
	for changed {
		changed = false
		for elem := range visited {
			if recipeMap[elem] != nil {
				continue
			}

			for _, comb := range combinations[elem] {
				if tierMap[comb.Left] < tierMap[elem] && tierMap[comb.Right] < tierMap[elem] {
					leftRecipe := recipeMap[comb.Left]
					rightRecipe := recipeMap[comb.Right]
					if leftRecipe != nil && rightRecipe != nil {
						fmt.Printf("Found recipe for %s: %s + %s\n", 
							elem, comb.Left, comb.Right)
						recipeMap[elem] = &Node{
							Element: elem,
							Left:    leftRecipe,
							Right:   rightRecipe,
						}
						changed = true
						break
					}
				}
			}
		}
	}

	result := recipeMap[target]
	if result != nil {
		fmt.Printf("\nSuccessfully found recipe for %s\n", target)
	} else {
		fmt.Printf("\nNo valid recipe found for %s\n", target)
	}
	return result
}

func FindRecipeDFS(target string, visited map[string]bool) *Node {
	if _, exists := combinations[target]; !exists && !isBasic(target) {
		return nil
	}

	if isBasic(target) {
		DFSVisitedCount++
		return &Node{Element: target}
	}

	if visited == nil {
		visited = make(map[string]bool)
		DFSVisitedCount = 0
	}

	if visited[target] {
		return nil
	}

	visited[target] = true
	DFSVisitedCount++
	defer func() { visited[target] = false }()

	for _, comb := range combinations[target] {
		if tierMap[comb.Left] < tierMap[target] && tierMap[comb.Right] < tierMap[target] {
			left := FindRecipeDFS(comb.Left, visited)
			if left == nil {
				continue
			}
			right := FindRecipeDFS(comb.Right, visited)
			if right != nil {
				return &Node{Element: target, Left: left, Right: right}
			}
		}
	}

	return nil
}

func FindMultipleRecipesDFS(target string) []*Node {
	fmt.Printf("\n=== Starting Multiple DFS search for: %s ===\n", target)
	
	if isBasic(target) {
		fmt.Printf("Found basic element: %s\n", target)
		DFSVisitedCount = 1
		return []*Node{{Element: target}}
	}

	if _, exists := combinations[target]; !exists {
		fmt.Printf("Element %s not found in combinations\n", target)
		DFSVisitedCount = 0
		return nil
	}

	visited := make(map[string]bool)
	recipeMap := make(map[string][]*Node)
	DFSVisitedCount = 0

	var findRecipes func(elem string) []*Node
	findRecipes = func(elem string) []*Node {
		if isBasic(elem) {
			DFSVisitedCount++
			return []*Node{{Element: elem}}
		}

		if visited[elem] {
			return nil
		}

		visited[elem] = true
		DFSVisitedCount++
		defer func() { visited[elem] = false }()

		if recipes, exists := recipeMap[elem]; exists {
			return recipes
		}

		var recipes []*Node
		for _, comb := range combinations[elem] {
			if tierMap[comb.Left] < tierMap[elem] && tierMap[comb.Right] < tierMap[elem] {
				leftRecipes := findRecipes(comb.Left)
				if len(leftRecipes) == 0 {
					continue
				}
				rightRecipes := findRecipes(comb.Right)
				if len(rightRecipes) == 0 {
					continue
				}

				for _, left := range leftRecipes {
					for _, right := range rightRecipes {
						fmt.Printf("Found recipe for %s: %s + %s\n", 
							elem, comb.Left, comb.Right)
						recipes = append(recipes, &Node{
							Element: elem,
							Left:    left,
							Right:   right,
						})
					}
				}
			}
		}

		recipeMap[elem] = recipes
		return recipes
	}

	results := findRecipes(target)
	if len(results) > 0 {
		fmt.Printf("\nSuccessfully found %d recipes for %s\n", len(results), target)
	} else {
		fmt.Printf("\nNo valid recipe found for %s\n", target)
	}
	return results
}

func FindMultipleRecipesBidirectional(target string) []*Node {
	fmt.Printf("\n=== Starting Multiple Bidirectional Search ===\n")
	fmt.Printf("Target: %s (Tier: %d)\n", target, tierMap[target])
	basics := getSortedBasicElements()
	fmt.Printf("Start Elements: %v\n", basics)

	if isBasic(target) {
		fmt.Printf("Target is a basic element, returning direct node\n")
		return []*Node{{Element: target}}
	}
	if _, exists := combinations[target]; !exists {
		fmt.Printf("Target element not found in combinations\n")
		return nil
	}

	forwardVisited := make(map[string][]*Node)
	forwardQueue := []string{}
	for _, b := range basics {
		forwardVisited[b] = []*Node{{Element: b}}
		forwardQueue = append(forwardQueue, b)
	}
	
	backwardVisited := make(map[string]bool)
	backwardQueue := []string{target}
	backwardVisited[target] = true
	
	BidirectionalVisitedCount = len(basics) + 1
	fmt.Printf("Initialized bidirectional search\n")

	var results []*Node
	for len(forwardQueue) > 0 && len(backwardQueue) > 0 {
		currentForward := forwardQueue[0]
		forwardQueue = forwardQueue[1:]
		fmt.Printf("\nForward exploring from: %s (Tier: %d)\n", currentForward, tierMap[currentForward])

		if backwardVisited[currentForward] {
			fmt.Printf("Found intersection at: %s\n", currentForward)
			forwardPaths := forwardVisited[currentForward]
			
			for _, comb := range combinations[target] {
				if comb.Left == currentForward || comb.Right == currentForward {
					var otherElement string
					if comb.Left == currentForward {
						otherElement = comb.Right
					} else {
						otherElement = comb.Left
					}
					
					if otherNodes, exists := forwardVisited[otherElement]; exists {
						for _, forwardPath := range forwardPaths {
							for _, otherNode := range otherNodes {
								results = append(results, &Node{
									Element: target,
									Left:   forwardPath,
									Right:  otherNode,
								})
							}
						}
					}
				}
			}
		}

		for _, comb := range combinations {
			for _, c := range comb {
				if (c.Left == currentForward || c.Right == currentForward) &&
					len(forwardVisited[c.Left]) > 0 && len(forwardVisited[c.Right]) > 0 &&
					tierMap[c.Left] < tierMap[c.Root] && tierMap[c.Right] < tierMap[c.Root] {
					
					if _, exists := forwardVisited[c.Root]; !exists {
						fmt.Printf("  Forward found: %s + %s = %s\n", c.Left, c.Right, c.Root)
						for _, left := range forwardVisited[c.Left] {
							for _, right := range forwardVisited[c.Right] {
								forwardVisited[c.Root] = append(forwardVisited[c.Root], &Node{
									Element: c.Root,
									Left:    left,
									Right:   right,
								})
							}
						}
						forwardQueue = append(forwardQueue, c.Root)
						BidirectionalVisitedCount++
					}
				}
			}
		}

		currentBackward := backwardQueue[0]
		backwardQueue = backwardQueue[1:]
		fmt.Printf("\nBackward exploring from: %s (Tier: %d)\n", currentBackward, tierMap[currentBackward])

		for _, comb := range combinations {
			for _, c := range comb {
				if c.Root == currentBackward {
					if !backwardVisited[c.Left] {
						backwardVisited[c.Left] = true
						backwardQueue = append(backwardQueue, c.Left)
						BidirectionalVisitedCount++
					}
					if !backwardVisited[c.Right] {
						backwardVisited[c.Right] = true
						backwardQueue = append(backwardQueue, c.Right)
						BidirectionalVisitedCount++
					}
				}
			}
		}
	}

	if len(results) > 0 {
		fmt.Printf("\nSuccessfully found %d recipes for %s\n", len(results), target)
	} else {
		fmt.Printf("\nNo valid recipe found for %s\n", target)
	}
	return results
}

func FindMultipleRecipes(target string, maxCount int, algorithm string) []*Node {
	if isBasic(target) {
		atomic.StoreInt32(&MultiVisitedCount, 1)
		return []*Node{{Element: target}}
	}

	if _, exists := combinations[target]; !exists {
		return nil
	}
	atomic.StoreInt32(&MultiVisitedCount, 0)
	var results []*Node
	var mu sync.Mutex

	recipeCache := sync.Map{}
	seen := sync.Map{}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	var findRecipeWithAlgorithm func(elem string) []*Node
	switch algorithm {
	case "bfs":
		findRecipeWithAlgorithm = func(elem string) []*Node {
			if nodes := FindMultipleRecipesBFS(elem); nodes != nil {
				atomic.AddInt32(&MultiVisitedCount, int32(GetBFSVisited()))
				return nodes
			}
			return nil
		}
	case "dfs":
		findRecipeWithAlgorithm = func(elem string) []*Node {
			if nodes := FindMultipleRecipesDFS(elem); nodes != nil {
				atomic.AddInt32(&MultiVisitedCount, int32(GetDFSVisited()))
				return nodes
			}
			return nil
		}
	case "bidirectional":
		findRecipeWithAlgorithm = func(elem string) []*Node {
			if nodes := FindMultipleRecipesBidirectional(elem); nodes != nil {
				atomic.AddInt32(&MultiVisitedCount, int32(GetBidirectionalVisited()))
				return nodes
			}
			return nil
		}
	default:
		findRecipeWithAlgorithm = func(elem string) []*Node {
			if nodes := FindMultipleRecipesBFS(elem); nodes != nil {
				atomic.AddInt32(&MultiVisitedCount, int32(GetBFSVisited()))
				return nodes
			}
			return nil
		}
	}

	var findAllCombinations func(elem string) [][]Combination
	findAllCombinations = func(elem string) [][]Combination {
		var allCombos [][]Combination
		for _, c := range combinations[elem] {
			if tierMap[c.Left] < tierMap[elem] && tierMap[c.Right] < tierMap[elem] {
				allCombos = append(allCombos, []Combination{c})
			}
		}
		return allCombos
	}

	var findRecipe func(ctx context.Context, elem string, visited map[string]bool) []*Node
	findRecipe = func(ctx context.Context, elem string, visited map[string]bool) []*Node {
		select {
		case <-ctx.Done():
			return nil
		default:
		}

		if isBasic(elem) {
			atomic.AddInt32(&MultiVisitedCount, 1)
			return []*Node{{Element: elem}}
		}

		if visited == nil {
			visited = make(map[string]bool)
		}

		if visited[elem] {
			return nil
		}

		visited[elem] = true
		defer func() { visited[elem] = false }()

		if cached, ok := recipeCache.Load(elem); ok {
			return cached.([]*Node)
		}

		var localResults []*Node
	
		allCombos := findAllCombinations(elem)

		for _, comboGroup := range allCombos {
			for _, c := range comboGroup {
				leftRecipes := findRecipeWithAlgorithm(c.Left)
				if len(leftRecipes) == 0 {
					continue
				}
				rightRecipes := findRecipeWithAlgorithm(c.Right)
				if len(rightRecipes) == 0 {
					continue
				}

				for _, left := range leftRecipes {
					for _, right := range rightRecipes {
						node := &Node{Element: elem, Left: left, Right: right}
						if elem == target {
							signature := serializeTree(node)
							if _, exists := seen.LoadOrStore(signature, true); !exists {
								mu.Lock()
								if len(results) < maxCount {
									results = append(results, node)
								}
								mu.Unlock()
								if len(results) >= maxCount {
									cancel()
									return localResults
								}
							}
						}
						localResults = append(localResults, node)
						atomic.AddInt32(&MultiVisitedCount, 1)
					}
				}
			}
		}
		if len(localResults) > 0 {
			sort.Slice(localResults, func(i, j int) bool {
				return treeDepth(localResults[i]) < treeDepth(localResults[j])
			})
			recipeCache.Store(elem, localResults)
		}
		return localResults
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		findRecipe(ctx, target, nil)
	}()
	wg.Wait()

	if len(results) > 0 {
		sort.Slice(results, func(i, j int) bool {
			return treeDepth(results[i]) < treeDepth(results[j])
		})
	}
	return results
}

func FindMultipleRecipesBFS(target string) []*Node {
	fmt.Printf("\n=== Starting Multiple BFS search for: %s ===\n", target)
	
	if isBasic(target) {
		fmt.Printf("Found basic element: %s\n", target)
		BFSVisitedCount = 1
		return []*Node{{Element: target}}
	}

	if _, exists := combinations[target]; !exists {
		fmt.Printf("Element %s not found in combinations\n", target)
		BFSVisitedCount = 0
		return nil
	}

	fmt.Printf("Found %d combinations for %s\n", len(combinations[target]), target)
	visited := make(map[string]bool)
	recipeMap := make(map[string][]*Node)
	queue := []string{target}
	BFSVisitedCount = 0

	fmt.Println("\nFirst pass: Collecting combinations...")
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true
		BFSVisitedCount++
		fmt.Printf("Visiting: %s (visited count: %d)\n", current, BFSVisitedCount)

		if isBasic(current) {
			fmt.Printf("Found basic element: %s\n", current)
			recipeMap[current] = []*Node{{Element: current}}
			continue
		}

		validCombos := []Combination{}
		for _, comb := range combinations[current] {
			if tierMap[comb.Left] < tierMap[current] && tierMap[comb.Right] < tierMap[current] {
				validCombos = append(validCombos, comb)
			}
		}

		sort.SliceStable(validCombos, func(i, j int) bool {
			iDiff := (tierMap[current] - tierMap[validCombos[i].Left]) + (tierMap[current] - tierMap[validCombos[i].Right])
			jDiff := (tierMap[current] - tierMap[validCombos[j].Left]) + (tierMap[current] - tierMap[validCombos[j].Right])
			return iDiff > jDiff
		})

		for _, comb := range validCombos {
			fmt.Printf("  Checking combination: %s (tier %d) + %s (tier %d) = %s (tier %d)\n", 
				comb.Left, tierMap[comb.Left], comb.Right, tierMap[comb.Right], comb.Root, comb.Tier)
			
			if !visited[comb.Left] {
				queue = append(queue, comb.Left)
				fmt.Printf("    Added to queue: %s\n", comb.Left)
			}
			if !visited[comb.Right] {
				queue = append(queue, comb.Right)
				fmt.Printf("    Added to queue: %s\n", comb.Right)
			}
		}
	}

	fmt.Println("\nSecond pass: Building recipes...")
	changed := true
	for changed {
		changed = false
		for elem := range visited {
			if len(recipeMap[elem]) > 0 {
				continue
			}

			for _, comb := range combinations[elem] {
				if tierMap[comb.Left] < tierMap[elem] && tierMap[comb.Right] < tierMap[elem] {
					leftRecipes := recipeMap[comb.Left]
					rightRecipes := recipeMap[comb.Right]
					if len(leftRecipes) > 0 && len(rightRecipes) > 0 {
						for _, left := range leftRecipes {
							for _, right := range rightRecipes {
								fmt.Printf("Found recipe for %s: %s + %s\n", 
									elem, comb.Left, comb.Right)
								recipeMap[elem] = append(recipeMap[elem], &Node{
									Element: elem,
									Left:    left,
									Right:   right,
								})
								changed = true
							}
						}
					}
				}
			}
		}
	}

	results := recipeMap[target]
	if len(results) > 0 {
		fmt.Printf("\nSuccessfully found %d recipes for %s\n", len(results), target)
	} else {
		fmt.Printf("\nNo valid recipe found for %s\n", target)
	}
	return results
}

func exploreRecipe(target string, visited map[string]bool, counter *int32, algorithm string) *Node {
	if isBasic(target) {
		atomic.AddInt32(counter, 1)
		return &Node{Element: target}
	}
	if visited[target] {
		return nil
	}
	visited[target] = true
	atomic.AddInt32(counter, 1)

	var candidates []*Node
	validCombos := []Combination{}

	targetTier := tierMap[target]

	for _, comb := range combinations[target] {
		if tierMap[comb.Left] < targetTier && tierMap[comb.Right] < targetTier {
			leftTierDiff := targetTier - tierMap[comb.Left]
			rightTierDiff := targetTier - tierMap[comb.Right]
			avgTierDiff := (leftTierDiff + rightTierDiff) / 2
			comb.Tier = avgTierDiff
			validCombos = append(validCombos, comb)
		}
	}

	sort.SliceStable(validCombos, func(i, j int) bool {
		if validCombos[i].Tier != validCombos[j].Tier {
			return validCombos[i].Tier < validCombos[j].Tier
		}
		return len(validCombos[i].Left)+len(validCombos[i].Right) < len(validCombos[j].Left)+len(validCombos[j].Right)
	})

	for _, comb := range validCombos {
		variations := []struct {
			left  string
			right string
		}{
			{comb.Left, comb.Right},
			{comb.Right, comb.Left},
		}

		for _, v := range variations {
			leftVisited := copyVisitedMap(visited)
			var left *Node
			
			switch algorithm {
			case "bfs":
				left = FindRecipeBFS(v.left)
			case "dfs":
				left = FindRecipeDFS(v.left, leftVisited)
			case "bidirectional":
				left = FindRecipeBidirectional(v.left)
			default:
				left = exploreRecipe(v.left, leftVisited, counter, algorithm)
			}

			if left == nil {
				continue
			}

			rightVisited := copyVisitedMap(visited)
			var right *Node
			
			switch algorithm {
			case "bfs":
				right = FindRecipeBFS(v.right)
			case "dfs":
				right = FindRecipeDFS(v.right, rightVisited)
			case "bidirectional":
				right = FindRecipeBidirectional(v.right)
			default:
				right = exploreRecipe(v.right, rightVisited, counter, algorithm)
			}

			if right == nil {
				continue
			}

			candidates = append(candidates,
				&Node{Element: target, Left: left, Right: right},
				&Node{Element: target, Left: right, Right: left},
			)
		}
	}

	if len(candidates) == 0 {
		return nil
	}

	rand.Seed(time.Now().UnixNano())
	return candidates[rand.Intn(len(candidates))]
}

func getSortedBasicElements() []string {
	basics := []string{}
	for elem := range tierMap {
		if isBasic(elem) {
			basics = append(basics, elem)
		}
	}
	sort.Strings(basics)
	return basics
}

func FindRecipeBidirectional(target string) *Node {
	fmt.Printf("\n=== Starting Bidirectional Search ===\n")
	fmt.Printf("Target: %s (Tier: %d)\n", target, tierMap[target])
	basics := getSortedBasicElements()
	fmt.Printf("Start Elements: %v\n", basics)

	if isBasic(target) {
		fmt.Printf("Target is a basic element, returning direct node\n")
		return &Node{Element: target}
	}
	if _, exists := combinations[target]; !exists {
		fmt.Printf("Target element not found in combinations\n")
		return nil
	}

	forwardVisited := make(map[string]*Node)
	forwardQueue := []string{}
	for _, b := range basics {
		forwardVisited[b] = &Node{Element: b}
		forwardQueue = append(forwardQueue, b)
	}
	
	backwardVisited := make(map[string]bool)
	backwardQueue := []string{target}
	backwardVisited[target] = true
	
	BidirectionalVisitedCount = len(basics) + 1
	fmt.Printf("Initialized bidirectional search\n")

	for len(forwardQueue) > 0 && len(backwardQueue) > 0 {
		currentForward := forwardQueue[0]
		forwardQueue = forwardQueue[1:]
		fmt.Printf("\nForward exploring from: %s (Tier: %d)\n", currentForward, tierMap[currentForward])

		if backwardVisited[currentForward] {
			fmt.Printf("Found intersection at: %s\n", currentForward)
			forwardPath := forwardVisited[currentForward]
			
			for _, comb := range combinations[target] {
				if comb.Left == currentForward || comb.Right == currentForward {
					var otherElement string
					if comb.Left == currentForward {
						otherElement = comb.Right
					} else {
						otherElement = comb.Left
					}
					
					if otherNode, exists := forwardVisited[otherElement]; exists {
						return &Node{
							Element: target,
							Left:   forwardPath,
							Right:  otherNode,
						}
					}
				}
			}
		}

		for _, comb := range combinations {
			for _, c := range comb {
				if (c.Left == currentForward || c.Right == currentForward) &&
					forwardVisited[c.Left] != nil && forwardVisited[c.Right] != nil &&
					tierMap[c.Left] < tierMap[c.Root] && tierMap[c.Right] < tierMap[c.Root] {
					
					if _, exists := forwardVisited[c.Root]; !exists {
						fmt.Printf("  Forward found: %s + %s = %s\n", c.Left, c.Right, c.Root)
						forwardVisited[c.Root] = &Node{
							Element: c.Root,
							Left:    forwardVisited[c.Left],
							Right:   forwardVisited[c.Right],
						}
						forwardQueue = append(forwardQueue, c.Root)
						BidirectionalVisitedCount++
					}
				}
			}
		}

		currentBackward := backwardQueue[0]
		backwardQueue = backwardQueue[1:]
		fmt.Printf("\nBackward exploring from: %s (Tier: %d)\n", currentBackward, tierMap[currentBackward])

		for _, comb := range combinations {
			for _, c := range comb {
				if c.Root == currentBackward {
					if !backwardVisited[c.Left] {
						backwardVisited[c.Left] = true
						backwardQueue = append(backwardQueue, c.Left)
						BidirectionalVisitedCount++
					}
					if !backwardVisited[c.Right] {
						backwardVisited[c.Right] = true
						backwardQueue = append(backwardQueue, c.Right)
						BidirectionalVisitedCount++
					}
				}
			}
		}
	}

	return nil
}

func copyVisitedMap(original map[string]bool) map[string]bool {
	copy := make(map[string]bool)
	for k, v := range original {
		copy[k] = v
	}
	return copy
}

func treeDepth(node *Node) int {
	if node == nil {
		return 0
	}
	leftDepth := treeDepth(node.Left)
	rightDepth := treeDepth(node.Right)
	if leftDepth > rightDepth {
		return leftDepth + 1
	}
	return rightDepth + 1
}

func serializeTree(n *Node) string {
	if n == nil {
		return ""
	}
	if n.Left == nil && n.Right == nil {
		return n.Element
	}
	leftStr := serializeTree(n.Left)
	rightStr := serializeTree(n.Right)
	if leftStr > rightStr {
		return n.Element + "(" + rightStr + "," + leftStr + ")"
	}
	return n.Element + "(" + leftStr + "," + rightStr + ")"
}

func IsBasic(element string) bool {
	return isBasic(element)
}

func GetCombinations(element string) []Combination {
	return combinations[element]
}

func IsLowerTier(c Combination) bool {
	return tierMap[c.Left] < tierMap[c.Root] && tierMap[c.Right] < tierMap[c.Root]
}

func GetBFSVisited() int {
	return BFSVisitedCount
}

func GetDFSVisited() int {
	return DFSVisitedCount
}

func GetMultiVisited() int {
	return int(atomic.LoadInt32(&MultiVisitedCount))
}

func GetBidirectionalVisited() int {
	return BidirectionalVisitedCount
}

func enableCORS(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next(w, r)
	}
}

func handleSearch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	element := r.URL.Query().Get("element")
	if element == "" {
		http.Error(w, "Element parameter is required", http.StatusBadRequest)
		return
	}

	mode := r.URL.Query().Get("mode")
	recipeMode := r.URL.Query().Get("recipe_mode")
	fmt.Printf("\n=== Search Request ===\n")
	fmt.Printf("Element: %s (Tier: %d)\n", element, tierMap[element])
	fmt.Printf("Mode: %s\n", mode)
	fmt.Printf("Recipe Mode: %s\n", recipeMode)

	var result *Node
	var results []*Node
	var visited int
	var executionTime time.Duration
	var response struct {
		Found         bool     `json:"found"`
		Steps         int      `json:"steps"`
		Paths         [][]Step `json:"paths"`
		Target        struct {
			Element string `json:"element"`
			Tier    int    `json:"tier"`
		} `json:"target"`
		ExecutionTime float64 `json:"executionTime"`
	}

	response.Target.Element = element
	response.Target.Tier = tierMap[element]

	startTime := time.Now()

	if recipeMode == "single" {
		switch mode {
		case "bfs":
			result = FindRecipeBFS(element)
			visited = GetBFSVisited()
			if result != nil {
				path := convertRecipeToPath(result)
				response.Found = true
				response.Steps = visited
				response.Paths = [][]Step{path}
			}
		case "dfs":
			result = FindRecipeDFS(element, nil)
			visited = GetDFSVisited()
			if result != nil {
				path := convertRecipeToPath(result)
				response.Found = true
				response.Steps = visited
				response.Paths = [][]Step{path}
			}
		case "bidirectional":
			result = FindRecipeBidirectional(element)
			visited = BidirectionalVisitedCount
			if result != nil {
				path := convertRecipeToPath(result)
				response.Found = true
				response.Steps = visited
				response.Paths = [][]Step{path}
			}
		default:
			fmt.Printf("Invalid mode: %s\n", mode)
			http.Error(w, "Invalid mode", http.StatusBadRequest)
			return
		}
	} else if recipeMode == "multiple" {
		maxRecipes := 10
		if maxRecipesStr := r.URL.Query().Get("max_recipes"); maxRecipesStr != "" {
			if parsed, err := strconv.Atoi(maxRecipesStr); err == nil && parsed > 0 {
				maxRecipes = parsed
			}
		}
		fmt.Printf("Requested max recipes: %d\n", maxRecipes)
		results = FindMultipleRecipes(element, maxRecipes, mode)
		visited = GetMultiVisited()
		if len(results) > 0 {
			paths := make([][]Step, 0, len(results))
			for _, result := range results {
				path := convertRecipeToPath(result)
				paths = append(paths, path)
			}
			response.Found = true
			response.Steps = visited
			response.Paths = paths
		}
	} else {
		http.Error(w, "Invalid recipe mode", http.StatusBadRequest)
		return
	}

	executionTime = time.Since(startTime)
	response.ExecutionTime = float64(executionTime.Microseconds()) / 1000.0

	w.Header().Set("Content-Type", "application/json")

	if err := json.NewEncoder(w).Encode(response); err != nil {
		fmt.Printf("Error encoding response: %v\n", err)
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

type Step struct {
	Ingredients []string `json:"ingredients"`
	Result      string   `json:"result"`
	Tiers       struct {
		Left   int `json:"left"`
		Right  int `json:"right"`
		Result int `json:"result"`
	} `json:"tiers"`
}

func convertRecipeToPath(node *Node) []Step {
	if node == nil {
		return nil
	}

	if node.Left == nil && node.Right == nil {
		return nil
	}

	leftSteps := convertRecipeToPath(node.Left)
	rightSteps := convertRecipeToPath(node.Right)

	currentStep := Step{
		Ingredients: []string{node.Left.Element, node.Right.Element},
		Result:      node.Element,
	}
	currentStep.Tiers.Left = tierMap[node.Left.Element]
	currentStep.Tiers.Right = tierMap[node.Right.Element]
	currentStep.Tiers.Result = tierMap[node.Element]

	steps := make([]Step, 0)
	steps = append(steps, leftSteps...)
	steps = append(steps, rightSteps...)
	steps = append(steps, currentStep)

	return steps
}

func handleMode(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	modes := []string{"bfs", "dfs", "bidirectional", "multi"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(modes)
}

func main() {
	fmt.Println("Starting server...")
	
	err := LoadCombinations("combinations.json")
	if err != nil {
		fmt.Printf("Error loading combinations: %v\n", err)
		panic(err)
	}

	http.HandleFunc("/search", enableCORS(handleSearch))
	http.HandleFunc("/mode", enableCORS(handleMode))

	port := ":5000"
	fmt.Printf("Server starting on port %s...\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		fmt.Printf("Error starting server: %v\n", err)
		panic(err)
	}
}