package ssdhelpers

import (
	"encoding/gob"
	"fmt"
	"math"
	"math/rand"
	"os"

	"github.com/pkg/errors"
	"github.com/semi-technologies/weaviate/adapters/repos/db/vector/hnsw/distancer"
)

type FilterFunc func([]float32) []float32

type KMeans struct {
	K                  int
	DeltaThreshold     float32
	IterationThreshold int
	Distance           distancer.Provider
	centers            [][]float32
	dimensions         int
	filter             FilterFunc

	data KMeansPartitionData
}

type KMeansPartitionData struct {
	changes      int
	points       []uint64
	cc           [][]uint64
	maxDistances []float32
	maxPoints    [][]float32
}

type KMeansData struct {
	K       int
	Centers [][]float32
}

const DataFileName = "kmeans.gob"

func NewKMeans(k int, distance distancer.Provider, dimensions int) *KMeans {
	kMeans := &KMeans{
		K:                  k,
		DeltaThreshold:     0.01,
		IterationThreshold: 10,
		Distance:           distance,
		dimensions:         dimensions,
		filter: func(x []float32) []float32 {
			return x
		},
	}
	return kMeans
}

func NewKMeansWithFilter(k int, distance distancer.Provider, dimensions int, filter FilterFunc) *KMeans {
	kMeans := NewKMeans(k, distance, dimensions)
	kMeans.filter = filter
	return kMeans
}

func (m *KMeans) ToDisk(path string, id int) {
	if m == nil {
		return
	}
	fData, err := os.Create(fmt.Sprintf("%s/%d.%s", path, id, DataFileName))
	if err != nil {
		panic(errors.Wrap(err, "Could not create kmeans file"))
	}
	defer fData.Close()

	dEnc := gob.NewEncoder(fData)
	err = dEnc.Encode(KMeansData{
		K:       m.K,
		Centers: m.centers,
	})
	if err != nil {
		panic(errors.Wrap(err, "Could not encode kmeans"))
	}
}

func KMeansFromDisk(path string, id int, distance distancer.Provider) *KMeans {
	fData, err := os.Open(fmt.Sprintf("%s/%d.%s", path, id, DataFileName))
	if err != nil {
		return nil
	}
	defer fData.Close()

	data := KMeansData{}
	dDec := gob.NewDecoder(fData)
	err = dDec.Decode(&data)
	if err != nil {
		panic(errors.Wrap(err, "Could not decode data"))
	}
	kmeans := NewKMeans(data.K, distance, 0)
	kmeans.centers = data.Centers
	return kmeans
}

func KMeansFromDiskWithFilter(path string, id int, distance distancer.Provider, filter FilterFunc) *KMeans {
	kmeans := KMeansFromDisk(path, id, distance)
	kmeans.filter = filter
	return kmeans
}

func (m *KMeans) Add(x []float32) {
	panic("Not implemented (KMeans.Add)")
}

func (m *KMeans) Centers() [][]float32 {
	return m.centers
}

func (m *KMeans) Encode(point []float32) byte {
	return byte(m.Nearest(point))
}

func (m *KMeans) Nearest(point []float32) uint64 {
	return m.NNearest(point, 1)[0]
}

func (m *KMeans) nNearest(point []float32, n int) ([]uint64, []float32) {
	mins := make([]uint64, n)
	minD := make([]float32, n)
	for i := range mins {
		mins[i] = 0
		minD[i] = math.MaxFloat32
	}
	for i, c := range m.centers {
		distance, _, _ := m.Distance.SingleDist(m.filter(point), c)
		j := 0
		for (j < n) && minD[j] < distance {
			j++
		}
		if j < n {
			for l := n - 1; l >= j+1; l-- {
				mins[l] = mins[l-1]
				minD[l] = minD[l-1]
			}
			minD[j] = distance
			mins[j] = uint64(i)
		}
	}
	return mins, minD
}

func (m *KMeans) NNearest(point []float32, n int) []uint64 {
	nearest, _ := m.nNearest(point, n)
	return nearest
}

func (m *KMeans) setCenters(centers [][]float32) {
	// ToDo: check size
	m.centers = centers
}

func (m *KMeans) initCenters(data [][]float32) {
	if m.centers != nil {
		return
	}
	m.centers = make([][]float32, 0, m.K)
	for i := 0; i < m.K; i++ {
		var vec []float32
		for vec == nil {
			vec = data[rand.Intn(len(data))]
		}
		m.centers = append(m.centers, m.filter(vec))
	}
}

func (m *KMeans) recluster(data [][]float32) {
	for p := 0; p < len(data); p++ {
		point := data[p]
		if point == nil {
			continue
		}
		cis, dis := m.nNearest(point, 1)
		ci, di := cis[0], dis[0]
		m.data.cc[ci] = append(m.data.cc[ci], uint64(p))
		if di > m.data.maxDistances[ci] {
			m.data.maxDistances[ci] = di
			m.data.maxPoints[ci] = m.filter(point)
		}
		if m.data.points[p] != ci {
			m.data.points[p] = ci
			m.data.changes++
		}
	}
}

func (m *KMeans) resortOnEmptySets(data [][]float32) {
	k64 := uint64(m.K)
	dataSize := len(data)
	for ci := uint64(0); ci < k64; ci++ {
		if len(m.data.cc[ci]) == 0 {
			var ri int
			for {
				ri = rand.Intn(dataSize)
				if data[ri] == nil {
					continue
				}
				if len(m.data.cc[m.data.points[ri]]) > 1 {
					break
				}
			}
			m.data.cc[ci] = append(m.data.cc[ci], uint64(ri))
			m.data.points[ri] = ci
			m.data.changes = dataSize
		}
	}
}

func (m *KMeans) recalcCenters(data [][]float32) {
	for index := 0; index < m.K; index++ {
		m.centers[index] = make([]float32, m.dimensions)
		for j := range m.centers[index] {
			m.centers[index][j] = 0
		}
		size := len(m.data.cc[index])
		for _, ci := range m.data.cc[index] {
			vec := data[ci]
			if vec == nil {
				panic("")
			}
			v := m.filter(vec)
			for j := 0; j < m.dimensions; j++ {
				m.centers[index][j] += v[j]
			}
		}
		for j := 0; j < m.dimensions; j++ {
			m.centers[index][j] /= float32(size)
		}
	}
}

func (m *KMeans) stopCondition(iterations int, dataSize int) bool {
	return iterations >= m.IterationThreshold ||
		m.data.changes < int(float32(dataSize)*m.DeltaThreshold)
}

func (m *KMeans) Fit(data [][]float32) error { // init centers using min/max per dimension
	m.initCenters(data)
	dataSize := len(data)
	m.data.points = make([]uint64, dataSize)
	m.data.changes = 1

	for i := 0; m.data.changes > 0; i++ {
		m.data.changes = 0
		m.data.cc = make([][]uint64, m.K)
		m.data.maxDistances = make([]float32, m.K)
		m.data.maxPoints = make([][]float32, m.K)
		for j := range m.data.cc {
			m.data.cc[j] = make([]uint64, 0)
		}

		m.recluster(data)
		m.resortOnEmptySets(data)
		if m.data.changes > 0 {
			m.recalcCenters(data)
		}

		if m.stopCondition(i, dataSize) {
			break
		}

	}

	return nil
}

func (m *KMeans) Center(point []float32) []float32 {
	return m.centers[m.Nearest(point)]
}

func (m *KMeans) Centroid(i byte) []float32 {
	return m.centers[i]
}