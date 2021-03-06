package support

import (
	"container/list"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"

	"github.com/evolbioinfo/gotree/io"
	"github.com/evolbioinfo/gotree/tree"
)

type BoosterSupporter struct {
	currentTree                    int
	mutex                          *sync.RWMutex
	stop                           bool
	silent                         bool
	computeMovedSpecies            bool
	computeTransferPerBranches     bool // All  transfered taxa per branches are computed
	computeHighTransferPerBranches bool // Only highest transfered taxa per branches are computed

	/* Cutoff for considering a branch : ok if norm distance to current
	bootstrap tree < cutoff (ex=0.05) => allows to compute a minimum depth also:
	norm_dist = distance / (p-1)
	=> If we want at least one species that moves (distance=1) at a given norm_dist cutoff, we need a depth p :
	p=(1/norm_dist)+1
	*/
	movedSpeciesCutoff float64
	// If false, then we do not normalize by the expected value: 1-avg/expected
	normalizeByExpected bool
}

// computeTransferPerBranches and computeHighTransferPerBranches are mutually expclusive
// computeTransferPerBranches has priority over computeHighTransferPerBranches
func NewBoosterSupporter(silent, computeMovedSpecies, computeTransferPerBranches, computeHighTransferPerBranches bool, movedSpeciesCutoff float64, normalizeByExpected bool) *BoosterSupporter {
	if movedSpeciesCutoff < 0 {
		movedSpeciesCutoff = 1.0
	}
	if movedSpeciesCutoff > 1 {
		movedSpeciesCutoff = 1.0
	}

	return &BoosterSupporter{
		currentTree:                    0,
		mutex:                          &sync.RWMutex{},
		stop:                           false,
		silent:                         silent,
		computeMovedSpecies:            computeMovedSpecies,
		movedSpeciesCutoff:             movedSpeciesCutoff,
		normalizeByExpected:            normalizeByExpected,
		computeTransferPerBranches:     computeTransferPerBranches,
		computeHighTransferPerBranches: computeHighTransferPerBranches,
	}
}

func (supporter *BoosterSupporter) ExpectedRandValues(depth int) float64 {
	return float64(depth - 1)
}

func (supporter *BoosterSupporter) NormalizeByExpected() bool {
	return supporter.normalizeByExpected
}

func (supporter *BoosterSupporter) NewBootTreeComputed() {
	supporter.mutex.Lock()
	supporter.currentTree++
	supporter.mutex.Unlock()
}

func (supporter *BoosterSupporter) Progress() int {
	supporter.mutex.RLock()
	defer supporter.mutex.RUnlock()
	return supporter.currentTree
}
func (supporter *BoosterSupporter) PrintMovingTaxa() bool {
	return supporter.computeMovedSpecies
}

func (supporter *BoosterSupporter) PrintTaxPerBranches() bool {
	return supporter.computeTransferPerBranches
}

func (supporter *BoosterSupporter) PrintHighTaxPerBranches() bool {
	return supporter.computeHighTransferPerBranches
}

func (supporter *BoosterSupporter) Cancel() {
	supporter.stop = true
}
func (supporter *BoosterSupporter) Canceled() bool {
	return supporter.stop
}

func (supporter *BoosterSupporter) Init(maxdepth int, nbtips int) {
	supporter.stop = false
	supporter.mutex = &sync.RWMutex{}
	supporter.currentTree = 0
}

func Update_all_i_c_post_order_ref_tree(refTree *tree.Tree, edges *[]*tree.Edge, bootTree *tree.Tree, bootEdges *[]*tree.Edge, i_matrix *[][]uint16, c_matrix *[][]uint16) {
	for i, child := range refTree.Root().Neigh() {
		update_i_c_post_order_ref_tree(refTree, edges, child, i, refTree.Root(), bootTree, bootEdges, i_matrix, c_matrix)
	}
}

// this function does the post-order traversal (recursive from the pseudoroot to the leaves, updating knowledge for the subtrees)
//   of the reference tree, examining only leaves (terminal edges) of the bootstrap tree.
//   It sends a probe from the orig node to the target node (nodes in ref_tree), calculating I_ij and C_ij
//   (see Brehelin, Gascuel, Martin 2008).
func update_i_c_post_order_ref_tree(refTree *tree.Tree, edges *[]*tree.Edge,
	current *tree.Node, curidx int, prev *tree.Node,
	bootTree *tree.Tree, bootEdges *[]*tree.Edge,
	i_matrix *[][]uint16, c_matrix *[][]uint16) {

	var e, be, e2 *tree.Edge
	var child *tree.Node
	var edge_id, edge_id2, be_id, k int

	e = prev.Edges()[curidx]
	edge_id = e.Id() /* all this is in ref_tree */

	if current.Tip() {
		for be_id, be = range *bootEdges { // for all the terminal edges of boot_tree
			if !be.Right().Tip() {
				continue
			}
			/* we only want to scan terminal edges of boot_tree, where the right son is a leaf */
			/* else we update all the I_ij and C_ij with i = edge_id */
			if current.Name() != be.Right().Name() {
				/* here the taxa are different */
				(*i_matrix)[edge_id][be_id] = 0
				(*c_matrix)[edge_id][be_id] = 1
			} else {
				/* same taxa here in T_ref and T_boot */
				(*i_matrix)[edge_id][be_id] = 1
				(*c_matrix)[edge_id][be_id] = 0
			}
		} /* end for on all edges of T_boot, for my_br being terminal */
	} else {
		/* now the case where my_br is not a terminal edge */
		/* first initialise (zero) the cells we are going to update */
		for be_id, be = range *bootEdges {
			// We initialize the i and c matrices for the edge edge_id with :
			// 	* 0 for i : because afterwards we do i[edge_id] = i[edge_id] || i[edge_id2]
			// 	* 1 for c : because afterwards we do c[edge_id] = c[edge_id] && c[edge_id2]
			if be.Right().Tip() {
				(*i_matrix)[edge_id][be_id] = 0
				(*c_matrix)[edge_id][be_id] = 1
			}
		}

		for k, child = range current.Neigh() {
			if child != prev {
				e2 = current.Edges()[k]
				edge_id2 = e2.Id()
				update_i_c_post_order_ref_tree(refTree, edges, child, k, current, bootTree, bootEdges, i_matrix, c_matrix)

				for be_id, be = range *bootEdges { /* for all the terminal edges of boot_tree */
					if !be.Right().Tip() {
						continue
					}

					// OR between two integers, result is 0 or 1 */
					if (*i_matrix)[edge_id][be_id] != 0 || (*i_matrix)[edge_id2][be_id] != 0 {
						(*i_matrix)[edge_id][be_id] = 1
					} else {
						(*i_matrix)[edge_id][be_id] = 0
					}

					// AND between two integers, result is 0 or 1
					if (*c_matrix)[edge_id][be_id] != 0 && (*c_matrix)[edge_id2][be_id] != 0 {
						(*c_matrix)[edge_id][be_id] = 1
					} else {
						(*c_matrix)[edge_id][be_id] = 0
					}
				}
			}
		}
	}
}

func Update_all_i_c_post_order_boot_tree(refTree *tree.Tree, ntips uint, edges *[]*tree.Edge,
	bootTree *tree.Tree, bootEdges *[]*tree.Edge,
	i_matrix *[][]uint16, c_matrix *[][]uint16, hamming *[][]uint16, min_dist *[]uint16, min_dist_edge *[]int) error {
	for i, child := range bootTree.Root().Neigh() {
		update_i_c_post_order_boot_tree(refTree, ntips, edges, bootTree, bootEdges, child, i, bootTree.Root(), i_matrix, c_matrix, hamming, min_dist, min_dist_edge)
	}

	/* and then some checks to make sure everything went ok */
	for _, e := range *edges {
		if (*min_dist)[e.Id()] < 0 {
			er := errors.New("Min dist should be >= 0")
			io.LogError(er)
			return er
		}
		if e.Right().Tip() && (*min_dist)[e.Id()] != 0 {
			er := errors.New(fmt.Sprintf("any terminal edge should have an exact match in any bootstrap tree (%d)", (*min_dist)[e.Id()]))
			io.LogError(er)
			return er
		}
	}
	return nil
}

// here we implement the second part of the Brehelin/Gascuel/Martin algorithm:
//    post-order traversal of the bootstrap tree, and numerical recurrence.
// in this function, orig and target are nodes of boot_tree (aka T_boot).
// min_dist is an array whose size is equal to the number of edges in T_ref.
//    It gives for each edge of T_ref its min distance to a split in T_boot.
func update_i_c_post_order_boot_tree(refTree *tree.Tree, ntips uint, edges *[]*tree.Edge,
	bootTree *tree.Tree, bootEdges *[]*tree.Edge,
	current *tree.Node, curindex int, prev *tree.Node,
	i_matrix *[][]uint16, c_matrix *[][]uint16,
	hamming *[][]uint16, min_dist *[]uint16, min_dist_edge *[]int) {

	var e, e2, e3 *tree.Edge
	var edge_id, edge_id2, edge_id3, j int
	var child *tree.Node

	e = prev.Edges()[curindex]
	edge_id = e.Id()

	if !current.Tip() {
		// because nothing to do in the case where target is a leaf: intersection and union already ok.
		// otherwise, keep on posttraversing in all other directions

		// first initialise (zero) the cells we are going to update
		for edge_id3 = 0; edge_id3 < len(*edges); edge_id3++ {
			(*i_matrix)[edge_id3][edge_id] = 0
			(*c_matrix)[edge_id3][edge_id] = 0
		}

		for j, child = range current.Neigh() {
			if child != prev {
				e2 = current.Edges()[j]
				edge_id2 = e2.Id()
				update_i_c_post_order_boot_tree(refTree, ntips, edges, bootTree, bootEdges, child, j, current,
					i_matrix, c_matrix, hamming, min_dist, min_dist_edge)
				for edge_id3 = 0; edge_id3 < len(*edges); edge_id3++ { /* for all the edges of ref_tree */
					(*i_matrix)[edge_id3][edge_id] += (*i_matrix)[edge_id3][edge_id2]
					(*c_matrix)[edge_id3][edge_id] += (*c_matrix)[edge_id3][edge_id2]
				}
			}
		}
	}

	for edge_id3, e3 = range *edges { // for all the edges of ref_tree
		e3numtips, _ := e3.NumTipsRight()
		// at this point we can calculate in all cases (internal branch or not) the Hamming distance at [i][edge_id],
		(*hamming)[edge_id3][edge_id] = // card of union minus card of intersection
			uint16(e3numtips) + // #taxa in the cluster i of T_ref
				(*c_matrix)[edge_id3][edge_id] - // #taxa in cluster edge_id of T_boot BUT NOT in cluster i of T_ref
				(*i_matrix)[edge_id3][edge_id] // #taxa in the intersection of the two clusters

		/* NEW!! Let's immediately calculate the right distance, taking into account the fact that the true disance is min (dist, N-dist) */
		if (*hamming)[edge_id3][edge_id] > uint16(ntips)/2 { // floor value
			(*hamming)[edge_id3][edge_id] = uint16(ntips) - (*hamming)[edge_id3][edge_id]
		}

		/*   and update the min of all Hamming (Transfer) distances hamming[i][j] over all j */
		if (*hamming)[edge_id3][edge_id] < (*min_dist)[edge_id3] {
			(*min_dist)[edge_id3] = (*hamming)[edge_id3][edge_id]
			(*min_dist_edge)[edge_id3] = edge_id
		}
	}
}

// Thread that takes bootstrap trees from the channel,
// computes the transfer dist for each edges of the ref tree
// and send it to the result channel
// At the end, returns the number of treated trees
func (supporter *BoosterSupporter) ComputeValue(refTree *tree.Tree, cpu int, edges []*tree.Edge,
	bootTreeChannel <-chan tree.Trees, valChan chan<- bootval, speciesChannel chan<- speciesmoved,
	taxPerBranchChannel chan<- []*list.List) error {
	tips := refTree.Tips()
	var min_dist []uint16 = make([]uint16, len(edges))
	var min_dist_edge []int = make([]int, len(edges))
	var i_matrix [][]uint16 = make([][]uint16, len(edges))
	var c_matrix [][]uint16 = make([][]uint16, len(edges))
	var hamming [][]uint16 = make([][]uint16, len(edges))
	var movedSpecies []int = make([]int, len(tips))
	// List of moved species per reference branches
	// It is initialized at each new bootstrap tree
	// It is to the taxperbranch channel reciever
	// to clear all lists
	var taxaTransferedPerBranch []*list.List

	vals := make([]int, len(edges))
	// Number of branches that have a normalized similarity (1- (min_dist/(n-1)) to
	// bootstrap trees > 0.8
	var nb_branches_close int
	var err error
	for treeV := range bootTreeChannel {
		if treeV.Err != nil {
			io.LogError(treeV.Err)
			err = treeV.Err
		} else {
			treeV.Tree.ReinitIndexes()
			err = refTree.CompareTipIndexes(treeV.Tree)

			if err == nil {
				nb_branches_close = 0
				if !supporter.silent {
					fmt.Fprintf(os.Stderr, "CPU : %02d - Bootstrap tree %d\r", cpu, treeV.Id)
				}
				bootEdges := treeV.Tree.Edges()
				taxaTransferedPerBranch = make([]*list.List, len(edges))
				for i, _ := range edges {
					min_dist[i] = uint16(len(tips))
					min_dist_edge[i] = -1
					if len(bootEdges) > len(i_matrix[i]) {
						i_matrix[i] = make([]uint16, len(bootEdges))
						c_matrix[i] = make([]uint16, len(bootEdges))
						hamming[i] = make([]uint16, len(bootEdges))
					}
				}

				for i, e := range bootEdges {
					e.SetId(i)
				}

				Update_all_i_c_post_order_ref_tree(refTree, &edges, treeV.Tree, &bootEdges, &i_matrix, &c_matrix)
				Update_all_i_c_post_order_boot_tree(refTree, uint(len(tips)), &edges, treeV.Tree, &bootEdges, &i_matrix, &c_matrix, &hamming, &min_dist, &min_dist_edge)

				for i, e := range edges {
					if e.Right().Tip() {
						taxaTransferedPerBranch[i] = list.New()
						continue
					}
					vals[i] = int(min_dist[i])
					if supporter.computeMovedSpecies || supporter.computeTransferPerBranches || supporter.computeHighTransferPerBranches {
						td, err := e.TopoDepth()
						if err != nil {
							io.LogError(err)
							return err
						}
						be := bootEdges[min_dist_edge[i]]
						norm := float64(vals[i]) / (float64(td) - 1.0)
						mindepth := int(math.Ceil(1.0/supporter.movedSpeciesCutoff + 1.0))
						if sm, er := speciesToMove(e, be, vals[i]); er != nil {
							io.LogError(er)
							return er
						} else {
							if supporter.computeMovedSpecies && norm <= supporter.movedSpeciesCutoff && td >= mindepth {
								for e := sm.Front(); e != nil; e = e.Next() {
									movedSpecies[e.Value.(uint)]++
								}
								nb_branches_close++
							}
							if supporter.computeTransferPerBranches || supporter.computeHighTransferPerBranches {
								// The list of taxons that move around branch i in that bootstrap tree
								taxaTransferedPerBranch[i] = sm
							} else {
								sm.Init() // Clear List
							}
						}

					}
					valChan <- bootval{
						vals[i],
						i,
						false,
					}
				}

				if supporter.computeMovedSpecies {
					for sp, nb := range movedSpecies {
						speciesChannel <- speciesmoved{
							uint(sp),
							float64(nb) / float64(nb_branches_close),
						}
						movedSpecies[sp] = 0
					}
				}
				if supporter.computeTransferPerBranches || supporter.computeHighTransferPerBranches {
					taxPerBranchChannel <- taxaTransferedPerBranch
				}
				supporter.NewBootTreeComputed()
				if supporter.stop {
					break
				}
			}
		}
		treeV.Tree.Delete()
	}
	return err
}

func Booster(reftree *tree.Tree, boottrees <-chan tree.Trees, logfile *os.File, silent, computeMovedSpecies, computeTransferPerBranches, computeHighTransferPerBranches bool, movedSpeciedCutoff float64, normalizedByExpected bool, cpus int) error {
	var supporter *BoosterSupporter = NewBoosterSupporter(silent, computeMovedSpecies, computeTransferPerBranches, computeHighTransferPerBranches, movedSpeciedCutoff, normalizedByExpected)
	return ComputeSupport(reftree, boottrees, logfile, cpus, supporter)
}

// Returns the list of species to move to go from one branch to the other
// Its length should correspond to given dist
// If not, exit with an error
func speciesToMove(e, be *tree.Edge, dist int) (*list.List, error) {
	var i uint
	diff := list.New()
	equ := list.New()

	for i = 0; i < e.Bitset().Len(); i++ {
		if e.Bitset().Test(i) != be.Bitset().Test(i) {
			diff.PushBack(i)
		} else {
			equ.PushBack(i)
		}
	}
	if diff.Len() < equ.Len() {
		if diff.Len() != dist {
			er := errors.New(fmt.Sprintf("Length of moved species array (%d) is not equal to the minimum distance found (%d)", diff.Len(), dist))
			io.LogError(er)
			return nil, er
		}
		equ.Init()
		return diff, nil
	}
	if equ.Len() != dist {
		er := errors.New(fmt.Sprintf("Length of moved species array (%d) is not equal to the minimum distance found (%d)", equ.Len(), dist))
		io.LogError(er)
		return nil, er
	}
	diff.Init()
	return equ, nil
}

// This function writes on the child node name the string: "branch_id|avg_dist|depth"
// and removes support information from each branch
func ReformatAvgDistance(t *tree.Tree) {
	for i, e := range t.Edges() {
		if e.Support() != tree.NIL_SUPPORT {
			td, _ := e.TopoDepth()
			e.Right().SetName(fmt.Sprintf("%d|%s|%d", i, e.SupportString(), td))
			e.SetSupport(tree.NIL_SUPPORT)
		}
	}
}

// This function takes all branch support values (that are considered as average
// transfer distances over bootstrap trees), normalizes them by the depth and
// convert them to similarity, i.e:
//     1-avg_dist/(depth-1)
func NormalizeTransferDistancesByDepth(t *tree.Tree) {
	for _, e := range t.Edges() {
		avgdist := e.Support()
		if avgdist != tree.NIL_SUPPORT {
			td, _ := e.TopoDepth()
			e.SetSupport(float64(1) - avgdist/float64(td-1))
		}
	}
}
