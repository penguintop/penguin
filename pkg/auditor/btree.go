package auditor

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
)

// !!!!!!!!
// Binary Full Tree
// !!!!!!!!

type TreeNodeValue *[]byte

type TreeNode struct {
	PtrLeftSon  *TreeNode
	PtrRightSon *TreeNode
	Data        TreeNodeValue
}

func ParentHash(leftBytes []byte, rightBytes []byte) []byte {
	hash := sha256.New()
	hash.Write(leftBytes)
	hash.Write(rightBytes)
	return hash.Sum(nil)
}

// check if the size value is 2^n
func CheckSize(size int) bool {
	t := size
	for {
		if t == 1 {
			break
		}

		v := t % 2
		if v != 0 {
			return false
		}

		t = t / 2
	}
	return true
}

func BuildBTreeFromRetrievalAddresses(retrievalAddresses [][]byte) (*TreeNode, error) {
	if !CheckSize(len(retrievalAddresses)) {
		return nil, errors.New("BuildBTreeFromRetrievalAddresses: invalid retrievalAddresses size")
	}

	// if size == 1
	if len(retrievalAddresses) == 1 {
		pNode := new(TreeNode)
		pNode.PtrLeftSon = nil
		pNode.PtrRightSon = nil
		pNode.Data = &retrievalAddresses[0]
		// tree root node
		return pNode, nil
	}

	listNext := make([]*TreeNode, 0)

	// init listPrev
	for i := 0; i < len(retrievalAddresses); i = i + 2 {
		lVal := retrievalAddresses[i]
		rVal := retrievalAddresses[i+1]

		lNode := new(TreeNode)
		lNode.PtrLeftSon, lNode.PtrRightSon = nil, nil
		lNode.Data = &lVal

		rNode := new(TreeNode)
		rNode.PtrLeftSon, rNode.PtrRightSon = nil, nil
		rNode.Data = &rVal

		pVal := ParentHash(lVal, rVal)
		pNode := new(TreeNode)
		pNode.PtrLeftSon = lNode
		pNode.PtrRightSon = rNode
		pNode.Data = &pVal

		listNext = append(listNext, pNode)
	}

	for {
		listTemp := listNext
		listNext = make([]*TreeNode, 0)

		// if size == 1
		if len(listTemp) == 1 {
			// tree root node
			return listTemp[0], nil
		}

		for i := 0; i < len(listTemp); i = i + 2 {
			lVal := *listTemp[i].Data
			rVal := *listTemp[i+1].Data

			pVal := ParentHash(lVal, rVal)
			pNode := new(TreeNode)
			pNode.PtrLeftSon = listTemp[i]
			pNode.PtrRightSon = listTemp[i+1]
			pNode.Data = &pVal

			listNext = append(listNext, pNode)
		}
	}
}

func (t *TreeNode) GetNodeHash() []byte {
	return *t.Data
}

func (t *TreeNode) GetNodeHashHex() string {
	return hex.EncodeToString(t.GetNodeHash())
}

func (t *TreeNode) GetRootRelatedHashHex() (string, []string) {
	rootNode := t
	rootHashHex := rootNode.GetNodeHashHex()

	// if tree depth is 1
	if rootNode.PtrLeftSon == nil || rootNode.PtrRightSon == nil {
		return rootHashHex, nil
	}

	lNode := rootNode.PtrLeftSon
	rNode := rootNode.PtrRightSon

	return rootHashHex, []string{lNode.GetNodeHashHex(), rNode.GetNodeHashHex()}
}

func (t *TreeNode) GetPathWayHashHex(pathway uint64) (string, [][]string, string) {
	rootNode := t
	rootHashHex := rootNode.GetNodeHashHex()

	// if tree depth is 1
	if rootNode.PtrLeftSon == nil || rootNode.PtrRightSon == nil {
		return rootHashHex, nil, rootNode.GetNodeHashHex()
	}

	pathwayHexList := make([][]string, 0)
	tempNode := rootNode
	m := pathway

	for {
		// reach the lowest layer of tree
		if tempNode.PtrLeftSon == nil || tempNode.PtrRightSon == nil {
			return rootHashHex, pathwayHexList, tempNode.GetNodeHashHex()
		}

		lNode := tempNode.PtrLeftSon
		rNode := tempNode.PtrRightSon

		pathwayHexList = append(pathwayHexList, []string{lNode.GetNodeHashHex(), rNode.GetNodeHashHex()})

		direction := m % 2
		if direction == 0 {
			// left
			tempNode = tempNode.PtrLeftSon
		} else {
			// right
			tempNode = tempNode.PtrRightSon
		}

		m = m / 2
	}
}
