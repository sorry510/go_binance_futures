package main

import (
  "context"
  "encoding/json"
  "fmt"
  "os"
  "sort"
  "time"

  "github.com/beego/beego/v2/client/orm"
  "github.com/beego/beego/v2/core/config"
  _ "github.com/go-sql-driver/mysql"
  _ "go_binance_futures/bootstrap"
  "go_binance_futures/service/backtest"
  "go_binance_futures/service/historicalmarket"
)

const featureCount = 9
const minLeaf = 2000

var featureNames = [featureCount]string{
  "ret1_side","ret4_side","ret12_side","body_range_side","taker_imb_side",
  "quote_ratio_8h","range_ratio_8h","trade_ratio_8h","close_loc_12h_side",
}

type Template struct {
  Name string
  Technology json.RawMessage
  Strategy json.RawMessage
}

type Sample struct {
  X [featureCount]float64
  Reward float64
  Resolved bool
  Year int
  Sym int
}

type Split struct {
  Valid bool
  Feature int
  Threshold float64
  Score float64
}

type Node struct {
  Leaf bool
  Split Split
  Left *Node
  Right *Node
  N int
  Mean float64
}

func quantile(vals []float64, q float64) float64 {
  x:=append([]float64(nil), vals...)
  sort.Float64s(x)
  if len(x)==0 { return 0 }
  pos:=q*float64(len(x)-1)
  lo:=int(pos)
  hi:=lo
  if float64(lo)<pos { hi=lo+1 }
  if hi>=len(x) { hi=len(x)-1 }
  if hi==lo { return x[lo] }
  w:=pos-float64(lo)
  return x[lo]*(1-w)+x[hi]*w
}

func stats(samples []Sample, ids []int) (int,float64) {
  if len(ids)==0 { return 0,0 }
  sum:=0.0
  for _,id:=range ids { sum+=samples[id].Reward }
  return len(ids),sum/float64(len(ids))
}

func bestSplit(samples []Sample, ids []int, thresholds [featureCount][]float64) Split {
  best:=Split{}
  bestScore:=-1e300
  for f:=0; f<featureCount; f++ {
    for _,th:=range thresholds[f] {
      nl,nr:=0,0
      sl,sr:=0.0,0.0
      for _,id:=range ids {
        if samples[id].X[f] <= th { nl++; sl+=samples[id].Reward } else { nr++; sr+=samples[id].Reward }
      }
      if nl<minLeaf || nr<minLeaf { continue }
      score:=sl*sl/float64(nl)+sr*sr/float64(nr)
      if score>bestScore {
        bestScore=score
        best=Split{Valid:true,Feature:f,Threshold:th,Score:score}
      }
    }
  }
  return best
}

func buildTree(samples []Sample, ids []int, thresholds [featureCount][]float64, depth int) *Node {
  n,mean:=stats(samples,ids)
  node:=&Node{Leaf:true,N:n,Mean:mean}
  if depth>=2 { return node }
  sp:=bestSplit(samples,ids,thresholds)
  if !sp.Valid { return node }
  left:=make([]int,0,n)
  right:=make([]int,0,n)
  for _,id:=range ids {
    if samples[id].X[sp.Feature] <= sp.Threshold { left=append(left,id) } else { right=append(right,id) }
  }
  node.Leaf=false
  node.Split=sp
  node.Left=buildTree(samples,left,thresholds,depth+1)
  node.Right=buildTree(samples,right,thresholds,depth+1)
  return node
}

func assign(node *Node, x [featureCount]float64) *Node {
  cur:=node
  for !cur.Leaf {
    if x[cur.Split.Feature] <= cur.Split.Threshold { cur=cur.Left } else { cur=cur.Right }
  }
  return cur
}

type Cond struct { F int; Th float64; Left bool }

func collect(node *Node, path []Cond, selected map[*Node]bool, leafID *int) {
  if node.Leaf {
    *leafID = *leafID + 1
    mark:=""
    if node.Mean>=1.0 { selected[node]=true; mark=" SELECTED" }
    fmt.Printf("LEAF %d n=%d meanReward=%.6f%s path=",*leafID,node.N,node.Mean,mark)
    for _,c:=range path {
      op:=">"
      if c.Left { op="<=" }
      fmt.Printf(" %s%s%.8f",featureNames[c.F],op,c.Th)
    }
    fmt.Println()
    return
  }
  collect(node.Left,append(path,Cond{F:node.Split.Feature,Th:node.Split.Threshold,Left:true}),selected,leafID)
  collect(node.Right,append(path,Cond{F:node.Split.Feature,Th:node.Split.Threshold,Left:false}),selected,leafID)
}
