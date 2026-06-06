package simulation

import (
	"math"
	"math/rand"
)

// This file isolates the physically-grounded agronomic equations of chapter 2:
// extraterrestrial radiation, reference evapotranspiration (Hargreaves-Samani and
// the full FAO-56 Penman-Monteith), the AR(1) air-temperature generator, the gamma
// rainfall sampler, and the Crop Water Stress Index (CWSI).

const (
	solarConstant = 0.0820 // Gsc, MJ m^-2 min^-1
	albedo        = 0.23   // reference grass albedo
	stefanBoltz   = 4.903e-9
)

// extraterrestrialRadiation computes Ra (MJ m^-2 day^-1) from latitude and day of
// year following FAO-56 (Allen et al., 1998). Replaces the previously hard-coded Ra.
func extraterrestrialRadiation(latitudeDeg float64, dayOfYear int) float64 {
	phi := latitudeDeg * math.Pi / 180.0
	j := float64(dayOfYear)

	dr := 1.0 + 0.033*math.Cos(2.0*math.Pi/365.0*j)    // inverse relative earth-sun distance
	decl := 0.409 * math.Sin(2.0*math.Pi/365.0*j-1.39) // solar declination
	x := -math.Tan(phi) * math.Tan(decl)               // for sunset hour angle
	x = math.Max(-1, math.Min(1, x))                   // clamp for polar edge cases
	ws := math.Acos(x)                                 // sunset hour angle

	ra := (24.0 * 60.0 / math.Pi) * solarConstant * dr *
		(ws*math.Sin(phi)*math.Sin(decl) + math.Cos(phi)*math.Cos(decl)*math.Sin(ws))
	return math.Max(0, ra)
}

// saturationVaporPressure returns e0(T) in kPa.
func saturationVaporPressure(t float64) float64 {
	return 0.6108 * math.Exp(17.27*t/(t+237.3))
}

// slopeVaporPressure returns Delta in kPa/°C.
func slopeVaporPressure(tMean float64) float64 {
	es := saturationVaporPressure(tMean)
	return 4098.0 * es / math.Pow(tMean+237.3, 2)
}

// et0Hargreaves implements the simplified Hargreaves-Samani equation (eq. 2.2),
// now with Ra derived from latitude and day of year.
func et0Hargreaves(tMax, tMin, latitudeDeg float64, dayOfYear int) float64 {
	tMean := (tMax + tMin) / 2.0
	ra := extraterrestrialRadiation(latitudeDeg, dayOfYear)
	return 0.0023 * ra * (tMean + 17.8) * math.Sqrt(math.Max(0, tMax-tMin))
}

// et0PenmanMonteith implements the full FAO-56 Penman-Monteith equation (eq. 2.3),
// the extended configuration. Net radiation, wind and vapor-pressure deficit are
// estimated from the available temperature data and latitude/day-of-year.
func et0PenmanMonteith(tMax, tMin, latitudeDeg float64, dayOfYear int, u2 float64) float64 {
	tMean := (tMax + tMin) / 2.0

	// vapor pressures: actual ea estimated from tMin as dew point (FAO-56 fallback)
	esMax := saturationVaporPressure(tMax)
	esMin := saturationVaporPressure(tMin)
	es := (esMax + esMin) / 2.0
	ea := saturationVaporPressure(tMin)
	if ea > es {
		ea = es
	}

	delta := slopeVaporPressure(tMean)
	const atmPressure = 95.0 // kPa, ~500 m elevation (Cobán highlands)
	gamma := 0.000665 * atmPressure

	// radiation budget
	ra := extraterrestrialRadiation(latitudeDeg, dayOfYear)
	rso := (0.75 + 2e-5*500.0) * ra                     // clear-sky solar radiation
	rs := 0.16 * math.Sqrt(math.Max(0, tMax-tMin)) * ra // Hargreaves solar estimate (krs interior)
	if rs > rso {
		rs = rso
	}
	rns := (1.0 - albedo) * rs
	rsRatio := 1.0
	if rso > 0 {
		rsRatio = math.Min(1.0, rs/rso)
	}
	tMaxK := tMax + 273.16
	tMinK := tMin + 273.16
	rnl := stefanBoltz * ((math.Pow(tMaxK, 4) + math.Pow(tMinK, 4)) / 2.0) *
		(0.34 - 0.14*math.Sqrt(ea)) * (1.35*rsRatio - 0.35)
	rn := rns - rnl
	const g = 0.0 // daily soil heat flux ~ 0

	num := 0.408*delta*(rn-g) + gamma*(900.0/(tMean+273.0))*u2*(es-ea)
	den := delta + gamma*(1.0+0.34*u2)
	return math.Max(0, num/den)
}

// sampleGamma draws from Gamma(shape, scale) via Marsaglia-Tsang. Used for the
// stochastic daily rainfall depth on wet days (chapter 2.3.1).
func sampleGamma(rng *rand.Rand, shape, scale float64) float64 {
	if shape < 1.0 {
		// boost: Gamma(a) = Gamma(a+1) * U^(1/a)
		u := rng.Float64()
		return sampleGamma(rng, shape+1.0, scale) * math.Pow(u, 1.0/shape)
	}
	d := shape - 1.0/3.0
	c := 1.0 / math.Sqrt(9.0*d)
	for {
		x := rng.NormFloat64()
		v := 1.0 + c*x
		if v <= 0 {
			continue
		}
		v = v * v * v
		u := rng.Float64()
		if u < 1.0-0.0331*x*x*x*x {
			return d * v * scale
		}
		if math.Log(u) < 0.5*x*x+d*(1.0-v+math.Log(v)) {
			return d * v * scale
		}
	}
}

// cropWaterStressIndex implements the CWSI (eq. 2.10, Idso/Jackson). The canopy-air
// temperature differential is modelled from the water-stress coefficient and the
// vapor-pressure deficit baselines; the result is normalised to [0, 1].
func cropWaterStressIndex(ksWater, tAir float64) float64 {
	vpd := saturationVaporPressure(tAir) - saturationVaporPressure(tAir)*0.6 // ~40% RH deficit
	// non-water-stressed (lower) baseline depends on VPD (Idso 1981), upper = non-transpiring
	ll := 2.0 - 1.6*vpd
	ul := 4.0
	// modelled canopy warming: cooler when transpiring (Ks→1), warmer under stress
	dT := ll + (1.0-ksWater)*(ul-ll)
	cwsi := (dT - ll) / (ul - ll)
	return math.Max(0, math.Min(1, cwsi))
}
