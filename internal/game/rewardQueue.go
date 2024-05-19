package game

import (
	"container/heap"
	"time"
)

type Reward struct {
	money          int32
	activationTime time.Time
	index          int
	userId         int32
}

type RewardQueue []*Reward

func (pq RewardQueue) Len() int { return len(pq) }

func (pq RewardQueue) Less(i, j int) bool {
	return pq[i].activationTime.Before(pq[j].activationTime)
}

func (pq RewardQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}

func (pq *RewardQueue) Push(x any) {
	n := len(*pq)
	item := x.(*Reward)
	item.index = n
	*pq = append(*pq, item)
}

func (pq *RewardQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil  // avoid memory leak
	item.index = -1 // for safety
	*pq = old[0 : n-1]
	return item
}

func (pq *RewardQueue) Top() *Reward {
	return (*pq)[len(*pq)-1]
}

// update modifies the priority and value of an Item in the queue.
func (pq *RewardQueue) update(item *Reward, money int32, activationTime time.Time) {
	item.money = money
	item.activationTime = activationTime
	heap.Fix(pq, item.index)
}
