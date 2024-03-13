package main

import (
  "fmt"
//   "encoding/json"
  "log"
  "github.com/hyperledger/fabric-contract-api-go/contractapi"
  "math"
  "sync"
  "strconv"
)

type Point struct{
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type TM struct{
	Nodes []Point `json:"node"`
	Edges [][]int `json:"edge"`
	RobotCurrentLocationNode map[string]int `json:"robotnode"`
}
//key need string instead of int ,idk
//RobotCurrentLocationNode key: robotID, val: robot stand on which node



var numOfNode_TM int
//TM中node的數量
var r_fusion_threshold float64
// SmartContract provides functions for managing an Asset
type MapContract struct {
  contractapi.Contract
  topologicalMap TM
  mutex sync.RWMutex
}

func (mc *MapContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	// Create a new topological map with no edges.
	log.Println("Init begin")


	init_node := Point{X: 0.0, Y: 0.0}
	init_edge := []int{0,0}
	mc.topologicalMap.Nodes = append(mc.topologicalMap.Nodes, init_node)
	mc.topologicalMap.Edges = append(mc.topologicalMap.Edges, init_edge)
	mc.topologicalMap.RobotCurrentLocationNode= make(map[string]int)
	r_fusion_threshold= 0.52
	
	numOfNode_TM = 0
	log.Println("Init done")
	return nil
}
func (mc *MapContract) ResetTM(ctx contractapi.TransactionContextInterface)  error {
	log.Println("Reset")
	var initTM TM

	init_node := Point{X: 0.0, Y: 0.0}
	init_edge := []int{0,0}
	initTM.Nodes = append(initTM.Nodes, init_node)
	initTM.Edges = append(initTM.Edges, init_edge)

	mc.topologicalMap.Nodes = initTM.Nodes
	mc.topologicalMap.Edges = initTM.Edges
	mc.topologicalMap.RobotCurrentLocationNode= make(map[string]int)
	
	
	numOfNode_TM = 0
	log.Println("Reset done")
	return nil
}
func (mc *MapContract) Set_r_fusion(ctx contractapi.TransactionContextInterface, New_r float64) error {
	r_fusion_threshold= New_r
	fmt.Println("set r_fusion_threshold to ", r_fusion_threshold)
	return nil
}
func (mc *MapContract) NewRobotJoin(ctx contractapi.TransactionContextInterface, NewRobotID int) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	fmt.Println("NewRobotJoin start")
	fmt.Println("NewRobotID: ", NewRobotID)
	
	//update # robots
	str_robotID := strconv.Itoa(NewRobotID)
	mc.topologicalMap.RobotCurrentLocationNode[str_robotID]= 0
	

	fmt.Println("Currently number of robot ", len(mc.topologicalMap.RobotCurrentLocationNode))
	
	fmt.Println("topologicalMap", mc.topologicalMap)
	fmt.Println("NewRobotJoin Done")
	return nil
}

//return the pointID just added 
func (mc *MapContract) Update(ctx contractapi.TransactionContextInterface, x float64, y float64, robotID int) error {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	fmt.Println("Tm Update start")
	str_robotID:=  strconv.Itoa(robotID)

	
	//if robot new position close to existed node, just move the robot to node instead of create a new node
	for i := 0 ; i< len(mc.topologicalMap.Nodes); i++{
		dx := mc.topologicalMap.Nodes[i].X - x
		dy := mc.topologicalMap.Nodes[i].Y - y
		dis := math.Sqrt(dx*dx + dy*dy)
		// fmt.Println(dis)
		if dis < r_fusion_threshold{
			fmt.Println("node ", i ," and new node are close")
			
			newEdge := []int{mc.topologicalMap.RobotCurrentLocationNode[str_robotID], i}
			mc.topologicalMap.Edges = append(mc.topologicalMap.Edges, newEdge)
			
			mc.topologicalMap.RobotCurrentLocationNode[str_robotID]= i
			
			return nil
		}
		
	}
	


	// Add the new point to the topological map.
	numOfNode_TM++
	fmt.Println("curNodeID ", numOfNode_TM)
	newNode := Point{X: x, Y: y}
	mc.topologicalMap.Nodes = append(mc.topologicalMap.Nodes, newNode)
	fmt.Println("add a new vertex: ", newNode)
	// Add the edge to the TM 
	
	newEdge := []int{mc.topologicalMap.RobotCurrentLocationNode[str_robotID], numOfNode_TM}
	mc.topologicalMap.Edges = append(mc.topologicalMap.Edges, newEdge)
	fmt.Println("add a new edge: ", newEdge)
	
	mc.topologicalMap.RobotCurrentLocationNode[str_robotID]= numOfNode_TM
	

	fmt.Println("Tm Update Done")
	return nil
}

func (mc *MapContract) GetRobotMapNode(ctx contractapi.TransactionContextInterface, robotID int) (int, error) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	fmt.Println("GetRobotMapNode")

	str_robotID:=  strconv.Itoa(robotID)
	return mc.topologicalMap.RobotCurrentLocationNode[str_robotID], nil
}


func (mc *MapContract) GetTopoMap(ctx contractapi.TransactionContextInterface) (TM, error) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	fmt.Println("GetTopoMap start")
		

	fmt.Println("topologicalMap", mc.topologicalMap)
	fmt.Println("GetTopoMap done")
	return mc.topologicalMap, nil
}

func (mc *MapContract) GetShortestPath(ctx contractapi.TransactionContextInterface, curNodeID int, goalPointID int) ([]Point, error) {
	mc.mutex.Lock()
	defer mc.mutex.Unlock()
	fmt.Println("GetShortestPath start")

	//make edge to adjacency list 
	adjEdge := make(map[int][]int, len(mc.topologicalMap.Nodes))
    for _ , val := range mc.topologicalMap.Edges{
        adjEdge[val[0]] = append(adjEdge[val[0]], val[1])
        adjEdge[val[1]] = append(adjEdge[val[1]], val[0])
    }
	
	resPath, resPathNum := ShortestPathBFS(mc.topologicalMap.Nodes, adjEdge, curNodeID, goalPointID)
	fmt.Println("resPathNum", resPathNum)
	fmt.Println("GetShortestPath done")
	return resPath, nil
}

func main() {
	assetChaincode, err := contractapi.NewChaincode(&MapContract{})
	if err != nil {
	  log.Panicf("Error creating asset-transfer-basic chaincode: %v", err)
	}
  
	if err := assetChaincode.Start(); err != nil {
	  log.Panicf("Error starting asset-transfer-basic chaincode: %v", err)
	}
}



//delete a value from slice
func deleteValue(s []int, val int) []int {
    for i, v := range s {
        if v == val {
            return append(s[:i], s[i+1:]...)
        }
    }
    return s
}


// 使用BFS算法計算最短路径
//return []Point
func ShortestPathBFS(nodes []Point, edge map[int][]int, start int, end int) ([]Point, []int) {
	prev := make(map[int]int)
	visited := make(map[int]bool)
	queue := []int{start}
	visited[start] = true
	for len(queue) > 0 {
		v := queue[0]
		queue = queue[1:]
		for _, dest := range edge[v] {
			if !visited[dest] {
				visited[dest] = true
				prev[dest] = v
				queue = append(queue, dest)
			}
		}
		if visited[end] {
			break 
		}
	}

	
	path_number := []int{}
	path := []Point{}
	for u := end; u != start; u = prev[u] {
		path = append(path, nodes[u])
		path_number = append(path_number, u)
	}
	path = append(path, nodes[start])
	path_number = append(path_number, start)
	
	for i, j := 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
		path_number[i], path_number[j] = path_number[j], path_number[i]
	}

	return path[1:], path_number[1:]
}