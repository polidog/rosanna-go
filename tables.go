package rosanna

import "strings"

// AttributeName は JSX の属性名を HTML と SVG の属性名にします。表にない名前はそのまま返します。
func AttributeName(name string) string {
	if mapped, ok := attributeNames[name]; ok {
		return mapped
	}
	return name
}

var attributeNames = map[string]string{
	"acceptCharset":              "accept-charset",
	"accessKey":                  "accesskey",
	"allowFullScreen":            "allowfullscreen",
	"autoCapitalize":             "autocapitalize",
	"autoComplete":               "autocomplete",
	"autoCorrect":                "autocorrect",
	"autoFocus":                  "autofocus",
	"autoPlay":                   "autoplay",
	"autoSave":                   "autosave",
	"cellPadding":                "cellpadding",
	"cellSpacing":                "cellspacing",
	"charSet":                    "charset",
	"classID":                    "classid",
	"className":                  "class",
	"colSpan":                    "colspan",
	"contentEditable":            "contenteditable",
	"contextMenu":                "contextmenu",
	"controlsList":               "controlslist",
	"crossOrigin":                "crossorigin",
	"dateTime":                   "datetime",
	"dirName":                    "dirname",
	"encType":                    "enctype",
	"enterKeyHint":               "enterkeyhint",
	"fetchPriority":              "fetchpriority",
	"formAction":                 "formaction",
	"formEncType":                "formenctype",
	"formMethod":                 "formmethod",
	"formNoValidate":             "formnovalidate",
	"formTarget":                 "formtarget",
	"frameBorder":                "frameborder",
	"hrefLang":                   "hreflang",
	"htmlFor":                    "for",
	"httpEquiv":                  "http-equiv",
	"imageSizes":                 "imagesizes",
	"imageSrcSet":                "imagesrcset",
	"inputMode":                  "inputmode",
	"isMap":                      "ismap",
	"itemID":                     "itemid",
	"itemProp":                   "itemprop",
	"itemRef":                    "itemref",
	"itemScope":                  "itemscope",
	"itemType":                   "itemtype",
	"keyParams":                  "keyparams",
	"keyType":                    "keytype",
	"marginHeight":               "marginheight",
	"marginWidth":                "marginwidth",
	"maxLength":                  "maxlength",
	"mediaGroup":                 "mediagroup",
	"minLength":                  "minlength",
	"noModule":                   "nomodule",
	"noValidate":                 "novalidate",
	"playsInline":                "playsinline",
	"popoverTarget":              "popovertarget",
	"popoverTargetAction":        "popovertargetaction",
	"radioGroup":                 "radiogroup",
	"readOnly":                   "readonly",
	"referrerPolicy":             "referrerpolicy",
	"rowSpan":                    "rowspan",
	"spellCheck":                 "spellcheck",
	"srcDoc":                     "srcdoc",
	"srcLang":                    "srclang",
	"srcSet":                     "srcset",
	"tabIndex":                   "tabindex",
	"useMap":                     "usemap",
	"accentHeight":               "accent-height",
	"alignmentBaseline":          "alignment-baseline",
	"arabicForm":                 "arabic-form",
	"baselineShift":              "baseline-shift",
	"capHeight":                  "cap-height",
	"clipPath":                   "clip-path",
	"clipRule":                   "clip-rule",
	"colorInterpolation":         "color-interpolation",
	"colorInterpolationFilters":  "color-interpolation-filters",
	"colorProfile":               "color-profile",
	"colorRendering":             "color-rendering",
	"dominantBaseline":           "dominant-baseline",
	"enableBackground":           "enable-background",
	"fillOpacity":                "fill-opacity",
	"fillRule":                   "fill-rule",
	"floodColor":                 "flood-color",
	"floodOpacity":               "flood-opacity",
	"fontFamily":                 "font-family",
	"fontSize":                   "font-size",
	"fontSizeAdjust":             "font-size-adjust",
	"fontStretch":                "font-stretch",
	"fontStyle":                  "font-style",
	"fontVariant":                "font-variant",
	"fontWeight":                 "font-weight",
	"glyphName":                  "glyph-name",
	"glyphOrientationHorizontal": "glyph-orientation-horizontal",
	"glyphOrientationVertical":   "glyph-orientation-vertical",
	"horizAdvX":                  "horiz-adv-x",
	"horizOriginX":               "horiz-origin-x",
	"imageRendering":             "image-rendering",
	"letterSpacing":              "letter-spacing",
	"lightingColor":              "lighting-color",
	"markerEnd":                  "marker-end",
	"markerMid":                  "marker-mid",
	"markerStart":                "marker-start",
	"overlinePosition":           "overline-position",
	"overlineThickness":          "overline-thickness",
	"paintOrder":                 "paint-order",
	"panose1":                    "panose-1",
	"pointerEvents":              "pointer-events",
	"renderingIntent":            "rendering-intent",
	"shapeRendering":             "shape-rendering",
	"stopColor":                  "stop-color",
	"stopOpacity":                "stop-opacity",
	"strikethroughPosition":      "strikethrough-position",
	"strikethroughThickness":     "strikethrough-thickness",
	"strokeDasharray":            "stroke-dasharray",
	"strokeDashoffset":           "stroke-dashoffset",
	"strokeLinecap":              "stroke-linecap",
	"strokeLinejoin":             "stroke-linejoin",
	"strokeMiterlimit":           "stroke-miterlimit",
	"strokeOpacity":              "stroke-opacity",
	"strokeWidth":                "stroke-width",
	"textAnchor":                 "text-anchor",
	"textDecoration":             "text-decoration",
	"textRendering":              "text-rendering",
	"transformOrigin":            "transform-origin",
	"underlinePosition":          "underline-position",
	"underlineThickness":         "underline-thickness",
	"unicodeBidi":                "unicode-bidi",
	"unicodeRange":               "unicode-range",
	"unitsPerEm":                 "units-per-em",
	"vAlphabetic":                "v-alphabetic",
	"vHanging":                   "v-hanging",
	"vIdeographic":               "v-ideographic",
	"vMathematical":              "v-mathematical",
	"vectorEffect":               "vector-effect",
	"vertAdvY":                   "vert-adv-y",
	"vertOriginX":                "vert-origin-x",
	"vertOriginY":                "vert-origin-y",
	"wordSpacing":                "word-spacing",
	"writingMode":                "writing-mode",
	"xHeight":                    "x-height",
	"xlinkActuate":               "xlink:actuate",
	"xlinkArcrole":               "xlink:arcrole",
	"xlinkHref":                  "xlink:href",
	"xlinkRole":                  "xlink:role",
	"xlinkShow":                  "xlink:show",
	"xlinkTitle":                 "xlink:title",
	"xlinkType":                  "xlink:type",
	"xmlBase":                    "xml:base",
	"xmlLang":                    "xml:lang",
	"xmlSpace":                   "xml:space",
	"xmlnsXlink":                 "xmlns:xlink",
}

func isVoidTag(lowerTag string) bool {
	switch lowerTag {
	case "area", "base", "br", "col", "embed", "hr", "img", "input", "link", "meta", "param", "source", "track", "wbr":
		return true
	}
	return false
}

func isRawTextTag(lowerTag string) bool {
	return lowerTag == "script" || lowerTag == "style"
}

// 中身をテキストとして読み、自身の終了タグでしか閉じない要素。エスケープしない raw text がこの中にあると閉じられる
func isTextOnlyAncestor(lowerTag string) bool {
	switch lowerTag {
	case "noscript", "textarea", "title", "xmp", "iframe", "noembed", "noframes":
		return true
	}
	return false
}

func isURLAttribute(lowerName string) bool {
	switch lowerName {
	case "href", "src", "action", "formaction", "data", "poster", "background", "xlink:href":
		return true
	}
	return false
}

// true / false を値として書く属性
func isBooleanStringAttribute(lowerName string) bool {
	if strings.HasPrefix(lowerName, "aria-") {
		return true
	}
	switch lowerName {
	case "contenteditable", "draggable", "spellcheck":
		return true
	}
	return false
}

// アニメーション要素が対象の属性に入れる値
func isAnimationValueAttribute(lowerName string) bool {
	switch lowerName {
	case "to", "values", "from", "by":
		return true
	}
	return false
}

// ブラウザの URL パーサーと同じ前処理をするので "\tjavascript:" や "java\tscript:" も通さない
func hasDangerousScheme(url string) bool {
	normalized := strings.NewReplacer("\t", "", "\n", "", "\r", "").Replace(url)
	normalized = strings.TrimLeftFunc(normalized, func(r rune) bool { return r <= 0x20 })
	scheme, _, found := strings.Cut(normalized, ":")
	if !found {
		return false
	}
	switch strings.ToLower(scheme) {
	case "javascript", "vbscript", "data":
		return true
	}
	return false
}

// values は ";" 区切りで、後ろの値も時間が来れば適用される
func hasDangerousAnimationValue(value string) bool {
	for _, part := range strings.Split(value, ";") {
		if hasDangerousScheme(part) {
			return true
		}
	}
	return false
}
