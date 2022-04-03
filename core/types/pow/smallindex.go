package pow

type SmallIndex []int

func (this *SmallIndex) Has(i int) bool {
	for _, v := range *this {
		if v == i {
			return true
		}
	}
	return false
}
