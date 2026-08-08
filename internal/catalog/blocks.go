// Package catalog holds the server's static content: the lesson library,
// the practice problem bank, topic recommendations, and Snap & Solve
// options. None of this is derived from a photo — it mirrors the sample
// content originally authored in the Flutter app's MockTutiServerClient
// (tuti/lib/services/tuti-server/mock_tuti_server_client.dart), which
// remains the source of truth for wording and structure when this catalog
// is extended.
package catalog

import tutiv1 "tuti-server/internal/genproto/tutiv1"

func text(s string) *tutiv1.ContentBlock {
	return &tutiv1.ContentBlock{Block: &tutiv1.ContentBlock_Text{Text: &tutiv1.TextBlock{Text: s}}}
}

func mathBlock(expr string) *tutiv1.ContentBlock {
	return &tutiv1.ContentBlock{Block: &tutiv1.ContentBlock_Math{Math: &tutiv1.MathBlock{Expression: expr}}}
}

func step(number int32, label string, content ...*tutiv1.ContentBlock) *tutiv1.ContentBlock {
	return &tutiv1.ContentBlock{Block: &tutiv1.ContentBlock_Step{Step: &tutiv1.StepBlock{
		StepNumber: number,
		Label:      label,
		Content:    content,
	}}}
}

func graph2D(title string, xAxis, yAxis *tutiv1.AxisConfig, series ...*tutiv1.Series2D) *tutiv1.ContentBlock {
	return &tutiv1.ContentBlock{Block: &tutiv1.ContentBlock_Graph_2D{Graph_2D: &tutiv1.Graph2DBlock{
		Title:  title,
		Series: series,
		XAxis:  xAxis,
		YAxis:  yAxis,
	}}}
}

func geometry(title string, viewBox *tutiv1.ViewBox2D, elements ...*tutiv1.GeometryElement) *tutiv1.ContentBlock {
	return &tutiv1.ContentBlock{Block: &tutiv1.ContentBlock_Geometry{Geometry: &tutiv1.GeometryBlock{
		Title:    title,
		Elements: elements,
		ViewBox:  viewBox,
	}}}
}

func axis(label string, min, max float64) *tutiv1.AxisConfig {
	return &tutiv1.AxisConfig{Label: label, Min: min, Max: max}
}

func viewBox(x, y, width, height float64) *tutiv1.ViewBox2D {
	return &tutiv1.ViewBox2D{X: x, Y: y, Width: width, Height: height}
}

func lineSeries(label, color string, points []*tutiv1.Point2D, dashed bool) *tutiv1.Series2D {
	return &tutiv1.Series2D{Label: label, Color: color, Points: points, Dashed: dashed}
}

func segment(x1, y1, x2, y2 float64, color string) *tutiv1.GeometryElement {
	return &tutiv1.GeometryElement{Element: &tutiv1.GeometryElement_Segment{Segment: &tutiv1.GeoSegment{
		X1: x1, Y1: y1, X2: x2, Y2: y2, Color: color,
	}}}
}

func geoAngle(cx, cy, radius, startDeg, endDeg float64, color string) *tutiv1.GeometryElement {
	return &tutiv1.GeometryElement{Element: &tutiv1.GeometryElement_Angle{Angle: &tutiv1.GeoAngle{
		Cx: cx, Cy: cy, Radius: radius, StartDegrees: startDeg, EndDegrees: endDeg, Color: color,
	}}}
}

func geoLabel(x, y float64, text, color string) *tutiv1.GeometryElement {
	return &tutiv1.GeometryElement{Element: &tutiv1.GeometryElement_Label{Label: &tutiv1.GeoLabel{
		X: x, Y: y, Text: text, Color: color,
	}}}
}

// linePoints and parabolaPoints mirror the Dart mock's identical helpers
// (_linePoints delegates straight to _parabolaPoints there too) — sample
// f over [start, end] in steps of `step`, rounded to 4 decimal places.
func linePoints(start, end, step float64, f func(float64) float64) []*tutiv1.Point2D {
	return parabolaPoints(start, end, step, f)
}

func parabolaPoints(start, end, step float64, f func(float64) float64) []*tutiv1.Point2D {
	var pts []*tutiv1.Point2D
	for x := start; x <= end+step*0.01; x += step {
		pts = append(pts, &tutiv1.Point2D{
			X: roundTo4(x),
			Y: roundTo4(f(x)),
		})
	}
	return pts
}

func roundTo4(v float64) float64 {
	const scale = 10000
	if v < 0 {
		return -float64(int64(-v*scale+0.5)) / scale
	}
	return float64(int64(v*scale+0.5)) / scale
}
