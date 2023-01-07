package main

import "math"

const h2oMolarWeight = 18.01528    // g/mol
const idealGasR = 8.31446261815324 // J/(K*mol)
const celciusZero = 273.15         // K

// calcAbsHum calculates absolute humidity based on temperature and relative humidity.
// rH given in %, T in degr Celcius.
// based on Hiung 2018 parametrization; DOI: https://doi.org/10.1175/JAMC-D-17-0334.1
func calcAbsHum(rH, T float64) (absH float64) {
	// calculate water vapour pressure, with respect to water or ice depending on T
	var pH2O float64 // in Pa
	if T >= 0 {
		pH2O = math.Exp(34.494-(4924.99/(T+237.1))) / math.Pow((T+105), 1.57) * (rH / 100)
	} else {
		pH2O = math.Exp(43.494-(6545.8/(T+278))) / math.Pow((T+868), 2) * (rH / 100)
	}

	// now calculate number of mols per m^3 (V=1) as n=(p*V)/(R*T)
	n := pH2O / (idealGasR * (celciusZero + T))

	// and return result multiplied by H2O molar weight to get
	return n * h2oMolarWeight
}
