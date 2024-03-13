package main

import (
  "fmt"
  "encoding/json"
  "log"
  "github.com/hyperledger/fabric-contract-api-go/contractapi"
//   "errors"
  "strings"
  "math"
  "sync"
  "strconv"
)

type FrontierPoint struct{
	X float64 `json:"x"`
	Y float64 `json:"y"`
	PreNodeID int `json:"preID"`
	DisFromPreNode float64 `json:"distance"`
	Invalid bool `json:"invalid"`
}

type TaskList struct{
	GoalPoints []FrontierPoint `json:"goalpts"`
}


type Point struct{
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type TM struct{
	Nodes []Point `json:"node"`
	Edges [][]int `json:"edge"`
	RobotCurrentLocationNode map[string]int `json:"robotnode"`
}
//RobotCurrentLocationNode key: robotID, val: robot stand on which node

//frotier跟TM node有多近要被過濾
var r_threshold_with_TM_node float64
//同一輪中產生的frontier要多近才被過濾
var r_threshold_with_same_round_frontiers float64
//未探索的frontier跟新的fronteir要多近才會被替換
var r_threshold_old_to_new float64
//該地方已經有frontier就不要再生了
// var r_threshold_existed_frontier float64
var open_step_penalty float64


// SmartContract provides functions for managing an Asset
type TaskContract struct {
  contractapi.Contract
  tasks TaskList
  mutex sync.RWMutex
}

func (tc *TaskContract) InitLedger(ctx contractapi.TransactionContextInterface) error {
	// Create a new topological map with no edges.
	fmt.Println("Init begin")
	tc.tasks = TaskList{}
	r_threshold_with_TM_node = 1.6
	r_threshold_with_same_round_frontiers= 0.3
	
	r_threshold_old_to_new= 0.0
	open_step_penalty= 0.45

	fmt.Println("Init Done")
	return nil
}
func (tc *TaskContract) ResetTask(ctx contractapi.TransactionContextInterface) error {
	// Create a new topological map with no edges.
	fmt.Println("Reset begin")
	tc.tasks= TaskList{}
	
	fmt.Println("Reset Done")
	return nil
}

func (tc *TaskContract) Set_r_TM_node_filter(ctx contractapi.TransactionContextInterface, New_r float64) error {
	r_threshold_with_TM_node= New_r
	fmt.Println("set r_threshold_with_TM_node to ", r_threshold_with_TM_node)
	return nil
}

//改成point 陣列加進來
func (tc *TaskContract) AddTask(ctx contractapi.TransactionContextInterface, pointsJSON string) error {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	fmt.Println("AddTask begin")
	
	

	var FPts []FrontierPoint
    err := json.Unmarshal([]byte(pointsJSON), &FPts)
    if err != nil {
        return fmt.Errorf("failed to unmarshal input JSON: %v", err)
    }

	/*change preNode begin*/

	fmt.Println("get topo")
	//get topoMap
	response:= ctx.GetStub().InvokeChaincode("tmcc", [][]byte{[]byte("GetTopoMap")}, "")
	// fmt.Println("response", string(response.Payload))
	str_payload := string(response.Payload)
	input := strings.Replace(str_payload, "response ", "", 1)
	// fmt.Println("TM is :", input)
	var tm TM
	err = json.Unmarshal([]byte(input), &tm)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TM response: %v", err)
	}

	for i:= 0; i< len(FPts); i++{
		minDis := 99999.9 //min dis
		minNodeID := -1 //min dis
		FPt := Point{FPts[i].X, FPts[i].Y}
		for j:= 0; j< len(tm.Nodes); j++{			
			dis := cal_distance(FPt, tm.Nodes[j])
			/**mindis start*/
			if dis < minDis{
				minDis = dis
				minNodeID= j
			}
			/*mindis end*/
		}
		FPts[i].PreNodeID = minNodeID
		FPts[i].DisFromPreNode= minDis

		fmt.Println("Node ", i, "and pre Node ", minNodeID, "have dis ", minDis)
	}

	/*change preNode end*/

	// Add the new points to the topological map.
    for _, fPt := range FPts {
        newTask := FrontierPoint{X: fPt.X, Y: fPt.Y, PreNodeID: fPt.PreNodeID, DisFromPreNode: fPt.DisFromPreNode}
		fmt.Println("add a new task: ", newTask)
        tc.tasks.GoalPoints = append(tc.tasks.GoalPoints, newTask)
    }

	fmt.Println("currently, number of tasks", len(tc.tasks.GoalPoints))
	
	
	fmt.Println("AddTask done")
	return nil
}


func (tc *TaskContract) GetTask(ctx contractapi.TransactionContextInterface) ([]FrontierPoint, error) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	fmt.Println("GetTask begin")

	
	fmt.Println("GetTask done: ", tc.tasks.GoalPoints)
	return tc.tasks.GoalPoints, nil
}


//輸入多個點，輸出多個點
func (tc *TaskContract) Filter(ctx contractapi.TransactionContextInterface, pointsJSON string) ([]FrontierPoint, error) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	fmt.Println("Filter begin")
	

	//Filter need to do three things: 
	//1. In the same round of frontier point detection, if two points are too close, one is eliminated
	//2. If the new frointer is close to any vertex on the TM, the new frontier will be eliminated
	


	var FPts []FrontierPoint
    err := json.Unmarshal([]byte(pointsJSON), &FPts)
    if err != nil {
        return []FrontierPoint{}, fmt.Errorf("failed to unmarshal input JSON: %v", err)
    }

	
	
	//misson 1 
	fmt.Println(" mission 1")
	//remove close points
	var eliminatedFPts []FrontierPoint
	for i := 0; i < len(FPts); i++ {
		isClose := false
        for j := i + 1; j < len(FPts); j++ {
			dis := cal_distance(Point{FPts[j].X, FPts[j].Y}, Point{FPts[i].X, FPts[i].Y})
            if dis < r_threshold_with_same_round_frontiers {
                isClose= true
				fmt.Println("remove close points which is created at same round")
				break
            }
        }
		if isClose== false{
			eliminatedFPts = append(eliminatedFPts, FPts[i])
		}
    }
	

	//mission 2

	fmt.Println(" mission 2")
	fmt.Println("before eliminatedFPts len", len(eliminatedFPts))
	eliminatedFPts_copy := eliminatedFPts
	eliminatedFPts = []FrontierPoint{}

	fmt.Println("get topo")
	//get topoMap
	response:= ctx.GetStub().InvokeChaincode("tmcc", [][]byte{[]byte("GetTopoMap")}, "")
	// fmt.Println("response", string(response.Payload))
	str_payload := string(response.Payload)
	input := strings.Replace(str_payload, "response ", "", 1)
	// fmt.Println("TM is :", input)
	var tm TM
	err = json.Unmarshal([]byte(input), &tm)
	if err != nil {
		return []FrontierPoint{}, fmt.Errorf("failed to unmarshal TM response: %v", err)
	}
	// fmt.Println("r_TM_node_threshold", r_threshold_with_TM_node)
	for i:= 0; i< len(eliminatedFPts_copy); i++{
		isCloseToNode := false
		// dis_min:= 9999999.9
		cur_FPt := Point{eliminatedFPts_copy[i].X, eliminatedFPts_copy[i].Y}
		for j:=0; j< len(tm.Nodes); j++{
			//misson 2
			dis := cal_distance(cur_FPt, tm.Nodes[j])
			if dis <= r_threshold_with_TM_node{
				isCloseToNode= true
				fmt.Println("reject by TM node and new frontier are close")
				break
			}
		}

		if isCloseToNode == false{
			// eliminatedFPts_copy[i].DisFromPreNode= cal_distance(Point{eliminatedFPts_copy[i].X, eliminatedFPts_copy[i].Y}, tm.Nodes[eliminatedFPts_copy[i].PreNodeID])
			eliminatedFPts= append(eliminatedFPts, eliminatedFPts_copy[i])
		}
	}
	fmt.Println("after eliminatedFPts len", len(eliminatedFPts))



	fmt.Println("Filter done")
	return eliminatedFPts, nil

}

func (tc *TaskContract) UpdateTask(ctx contractapi.TransactionContextInterface) error {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	fmt.Println("UpdateTask begin")

	fmt.Println("get topo")
	//get topoMap
	response:= ctx.GetStub().InvokeChaincode("tmcc", [][]byte{[]byte("GetTopoMap")}, "")
	// fmt.Println("response", string(response.Payload))
	str_payload := string(response.Payload)
	input := strings.Replace(str_payload, "response ", "", 1)
	// fmt.Println("TM is :", input)
	var tm TM
	err := json.Unmarshal([]byte(input), &tm)
	if err != nil {
		return fmt.Errorf("failed to unmarshal TM response: %v", err)
	}

	fmt.Println("before eliminate len(tc.GoalPoints)", len(tc.tasks.GoalPoints))
	var eliminatedUnexploredFrontier []FrontierPoint
	for i:= 0; i< len(tc.tasks.GoalPoints); i++{
		minDis := 99999.9 //min dis
		minNodeID := -1 //min dis
		kill := false
		FPt := Point{tc.tasks.GoalPoints[i].X, tc.tasks.GoalPoints[i].Y}
		for j:= 0; j< len(tm.Nodes); j++{			
			dis := cal_distance(FPt, tm.Nodes[j])
			if dis < r_threshold_with_TM_node{
				kill = true
				break
			}
			/**mindis start*/
			if dis < minDis{
				minDis = dis
				minNodeID= j
			}
			/*mindis end*/
		}
		if kill == false{
			tc.tasks.GoalPoints[i].PreNodeID= minNodeID //minDis
			tc.tasks.GoalPoints[i].DisFromPreNode= minDis //minDis
			eliminatedUnexploredFrontier = append(eliminatedUnexploredFrontier, tc.tasks.GoalPoints[i])
		}
	}


	tc.tasks.GoalPoints= eliminatedUnexploredFrontier
	fmt.Println("after eliminate len(tc.GoalPoints)", len(tc.tasks.GoalPoints))
	return nil
}

func (tc *TaskContract) TaskAllocationGreedy(ctx contractapi.TransactionContextInterface, robotID int) (FrontierPoint, error) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	fmt.Println("TaskAllocationGreedy begin")
	fmt.Println("Robot ", robotID, "wants to do a task allocation")
	var valid_task []FrontierPoint
	//key : valid_task idx, val: tc.tasks.GoalPoints idx
	var idxMapTable = make(map[int]int)
	
	
	count := 0
	for i:=0; i< len(tc.tasks.GoalPoints); i++{
		if tc.tasks.GoalPoints[i].Invalid == false{
			valid_task = append(valid_task, tc.tasks.GoalPoints[i])
			idxMapTable[count]= i
			count++
		}
	}

	if len(valid_task) == 0{
		fmt.Println("Have not more taks currently")
		return FrontierPoint{PreNodeID: -87}, nil
	}

	fmt.Println("get topo")
	//get topoMap
	response:= ctx.GetStub().InvokeChaincode("tmcc", [][]byte{[]byte("GetTopoMap")}, "")
	// fmt.Println("response", string(response.Payload))
	str_payload := string(response.Payload)
	input := strings.Replace(str_payload, "response ", "", 1)
	// fmt.Println("TM is :", input)
	var tm TM
	err := json.Unmarshal([]byte(input), &tm)
	if err != nil {
		return FrontierPoint{}, fmt.Errorf("failed to unmarshal TM response: %v", err)
	}


	//make edge to adjacency list 
	adj := make(map[int][]int, len(tm.Nodes))
    for _ , val := range tm.Edges{
        adj[val[0]] = append(adj[val[0]], val[1])
        adj[val[1]] = append(adj[val[1]], val[0])
    }
	//cost Table initialize
	//costTable[i][j] denotes the cost of robot i going to node j
	numOfTask := len(valid_task)
	numOfRobot := len(tm.RobotCurrentLocationNode)
	fmt.Println("numOfTask: ", numOfTask, "numOfRobot: ", numOfRobot)
	costTable := make([][]int, numOfRobot)
    for i := range  costTable{
        costTable[i] = make([]int, numOfTask)
    }

	//distance penalty
	for j:= 0; j< len(costTable[0]); j++{
		penalty := int(valid_task[j].DisFromPreNode / open_step_penalty)
		for i:= 0; i< len(costTable); i++{			
			costTable[i][j] += penalty
		}
		// fmt.Println("frontier ", j, "has penalty ", penalty)
	}
	

	robot_located := tm.RobotCurrentLocationNode[strconv.Itoa(robotID)]

	//compute cost by bfs
	//bfs return cost array, array[0] denotes the cost between the frontier point and node 0 ...
	// fmt.Println("len(tm.Node) ", len(tm.Node))
	for ti, task := range valid_task{
		cost := bfs(task.PreNodeID, adj, len(tm.Nodes))
		costTable[robotID][ti] += cost[robot_located]
	
	}


	minCost := 999999
	minCostIdx := -1

	for i:= 0; i< len(valid_task); i++{
		if costTable[robotID][i] < minCost{
			minCost= costTable[robotID][i]
			minCostIdx= i
		}
	}

	var res FrontierPoint
	res = valid_task[minCostIdx]
	tc.tasks.GoalPoints[idxMapTable[minCostIdx]].Invalid= true
	return res, nil
}


//傳入機器人id而不是位置
func (tc *TaskContract) TaskAllocation(ctx contractapi.TransactionContextInterface, robotID int) (FrontierPoint, error) {
	tc.mutex.Lock()
	defer tc.mutex.Unlock()
	fmt.Println("TaskAllocation begin")
	fmt.Println("Robot ", robotID, "wants to do a task allocation")

	var valid_task []FrontierPoint
	//key : valid_task idx, val: tc.tasks.GoalPoints idx
	var idxMapTable = make(map[int]int)
	
	
	count := 0
	for i:=0; i< len(tc.tasks.GoalPoints); i++{
		if tc.tasks.GoalPoints[i].Invalid == false{
			valid_task = append(valid_task, tc.tasks.GoalPoints[i])
			idxMapTable[count]= i
			count++
		}
	}
	fmt.Println("all tasks :", len(tc.tasks.GoalPoints), "valid tasks", len(valid_task))

	if len(valid_task) == 0{
		fmt.Println("Have not more taks currently")
		return FrontierPoint{PreNodeID: -87}, nil
	}
	

	fmt.Println("get topo")
	//get topoMap
	response:= ctx.GetStub().InvokeChaincode("tmcc", [][]byte{[]byte("GetTopoMap")}, "")
	// fmt.Println("response", string(response.Payload))
	str_payload := string(response.Payload)
	input := strings.Replace(str_payload, "response ", "", 1)
	// fmt.Println("TM is :", input)
	var tm TM
	err := json.Unmarshal([]byte(input), &tm)
	if err != nil {
		return FrontierPoint{}, fmt.Errorf("failed to unmarshal TM response: %v", err)
	}
	// fmt.Println("tm.Node",tm.Nodes)
	// fmt.Println("tm.Edge",tm.Edges)
	// fmt.Println("Currently number of robots exist in map : ", tm.RobotNum)

	//make edge to adjacency list 
	adj := make(map[int][]int, len(tm.Nodes))
    for _ , val := range tm.Edges{
        adj[val[0]] = append(adj[val[0]], val[1])
        adj[val[1]] = append(adj[val[1]], val[0])
    }
	fmt.Println("adj init done")
	//cost Table initialize
	//costTable[i][j] denotes the cost of robot i going to node j
	numOfTask := len(valid_task)
	numOfRobot := len(tm.RobotCurrentLocationNode)
	fmt.Println("numOfTask: ", numOfTask, "numOfRobot: ", numOfRobot)
	costTable := make([][]int, numOfRobot)
    for i := range  costTable{
        costTable[i] = make([]int, numOfTask)
    }

	//distance penalty
	for j:= 0; j< len(costTable[0]); j++{
		penalty := int(valid_task[j].DisFromPreNode / open_step_penalty)
		for i:= 0; i< len(costTable); i++{			
			costTable[i][j] += penalty
		}
		// fmt.Println("frontier ", j, "has penalty ", penalty)
	}
	


	fmt.Println("costTable init done")
	//compute cost by bfs
	//bfs return cost array, array[0] denotes the cost between the frontier point and node 0 ...
	// fmt.Println("len(tm.Node) ", len(tm.Node))
	for ti, task := range valid_task{
		cost := bfs(task.PreNodeID, adj, len(tm.Nodes))

		for str_robot_id, located := range tm.RobotCurrentLocationNode{
			robot_id, err := strconv.Atoi(str_robot_id)
			if err != nil {
				fmt.Println("string robot_id covert to int robotID failed")
				return FrontierPoint{}, nil
			}
			costTable[robot_id][ti] += cost[located]
		}
	
	}

	//minPos
	diff := make([]int, numOfTask)
	for ti := 0; ti < numOfTask; ti++{
	    for ri := 0 ; ri< numOfRobot; ri++{
	        if  robotID != ri && costTable[ri][ti] < costTable[robotID][ti]{
	            diff[ti]++
	        }
	    }
	}

	// for i, val := range(diff){
	// 	fmt.Println("diff[", i, "]", " has val: ", val)
	// }
	// fmt.Println("diff ", diff)
	minPos := 9999999
	minPosIdx := 9999999

	for i:= 0; i< len(diff); i++{
		if diff[i]< minPos{
			minPos= diff[i]
			minPosIdx= i
		}else if diff[i] == minPos{
			if costTable[robotID][i] < costTable[robotID][minPosIdx]{
				minPosIdx= i
				fmt.Println("same Pij, select low cost on costTalbe")
			}
			
		}
	}

	// fmt.Println("minPosIdx ", minPosIdx)
	var resTask FrontierPoint
	resTask = valid_task[minPosIdx]
	// fmt.Println("resTask ", resTask)
	//set the frontier to invalid, prevent double execute
	
	tc.tasks.GoalPoints[idxMapTable[minPosIdx]].Invalid= true
	// fmt.Println("resTask from valid_task :", resTask, " resTask from tc.tasks:", tc.tasks.GoalPoints[idxMapTable[minPosIdx]])
	
	fmt.Println("resTask ID: ", idxMapTable[minPosIdx], "resTask ", resTask)
	
	fmt.Println("TaskAllocation done")
	return resTask, nil
}
//output: [distance1, distance2, ...] , idx 0 refers to the distance between node 0 and target s ...
func bfs(s int, adj map[int][]int, n int) []int {
    dist := make([]int, n)
    for i := range dist {
        dist[i] = -1 // 初始化-1
    }
    dist[s] = 0
    q := []int{s}
    for len(q) > 0 {
        u := q[0]
        q = q[1:]
        for _, v := range adj[u] {
            if dist[v] == -1 { // 未訪問過
                dist[v] = dist[u] + 1
                q = append(q, v)
            }
        }
    }
    return dist
}


func cal_distance(p1, p2 Point) float64{
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return math.Sqrt(dx*dx + dy*dy)
}
func main() {
	assetChaincode, err := contractapi.NewChaincode(&TaskContract{})
	if err != nil {
	  log.Panicf("Error creating asset-transfer-basic chaincode: %v", err)
	}
  
	if err := assetChaincode.Start(); err != nil {
	  log.Panicf("Error starting asset-transfer-basic chaincode: %v", err)
	}
}


