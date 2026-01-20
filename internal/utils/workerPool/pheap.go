package workerPool

import "wfts/internal/model"

type CrawlStream []model.CrawlNode

func (cs CrawlStream) Len() int { return len(cs) }
func (cs CrawlStream) Swap(i, j int) { cs[i], cs[j] = cs[j], cs[i] }

func (cs CrawlStream) Less(i, j int) bool {
	if cs[i].Depth != cs[j].Depth {
        return cs[i].Depth < cs[j].Depth
    }
    return cs[i].SameDomain && !cs[j].SameDomain
}

func (cs *CrawlStream) Push(x any) {
	*cs = append(*cs, x.(model.CrawlNode))
}

func (cs *CrawlStream) Pop() any {
	n := cs.Len()
	if n == 0 {
		return model.CrawlNode{}
	}
	old := *cs
	last := old[n - 1]
	*cs = old[:n - 1]
	return last
}