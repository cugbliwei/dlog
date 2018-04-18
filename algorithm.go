package dlog

import "math"

//文本相似度
func Similarity(a, b string) float64 {
	return float64(LCS(a, b)) / (float64(LD(a, b)) + float64(LCS(a, b)))
}

//编辑距离
func LD(a, b string) int {
	n := len(a)
	m := len(b)
	if n == 0 {
		return m
	} else if m == 0 {
		return n
	}

	c := [][]int{}
	for i := 0; i <= n; i++ {
		var tmp []int
		for j := 0; j <= m; j++ {
			tmp = append(tmp, 0)
		}
		c = append(c, tmp)
	}
	for i := 0; i <= n; i++ {
		c[i][0] = i
	}
	for i := 0; i <= m; i++ {
		c[0][i] = i
	}

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			var cost int
			if a[i-1:i] != b[j-1:j] {
				cost = 1
			}

			above := c[i-1][j] + 1
			left := c[i][j-1] + 1
			diag := c[i-1][j-1] + cost
			c[i][j] = int(math.Min(float64(above), math.Min(float64(left), float64(diag))))
		}
	}
	return c[n][m]
}

//最长公共子序列
func LCS(a, b string) int {
	n := len(a)
	m := len(b)
	c := [][]int{}
	for i := 0; i <= n; i++ {
		var tmp []int
		for j := 0; j <= m; j++ {
			tmp = append(tmp, 0)
		}
		c = append(c, tmp)
	}

	for i := 0; i <= n; i++ {
		for j := 0; j <= m; j++ {
			if i == 0 || j == 0 {
				c[i][j] = 0
			} else if a[i-1:i] == b[j-1:j] {
				c[i][j] = c[i-1][j-1] + 1
			} else if c[i-1][j] >= c[i][j-1] {
				c[i][j] = c[i-1][j]
			} else {
				c[i][j] = c[i][j-1]
			}
		}
	}
	return c[n][m]
}
