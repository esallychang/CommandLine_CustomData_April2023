// ===========================================================================
//
//                            PUBLIC DOMAIN NOTICE
//            National Center for Biotechnology Information (NCBI)
//
//  This software/database is a "United States Government Work" under the
//  terms of the United States Copyright Act. It was written as part of
//  the author's official duties as a United States Government employee and
//  thus cannot be copyrighted. This software/database is freely available
//  to the public for use. The National Library of Medicine and the U.S.
//  Government do not place any restriction on its use or reproduction.
//  We would, however, appreciate having the NCBI and the author cited in
//  any work or product based on this material.
//
//  Although all reasonable efforts have been taken to ensure the accuracy
//  and reliability of the software and data, the NLM and the U.S.
//  Government do not and cannot warrant the performance or results that
//  may be obtained by using this software or data. The NLM and the U.S.
//  Government disclaim all warranties, express or implied, including
//  warranties of performance, merchantability or fitness for any particular
//  purpose.
//
// ===========================================================================
//
// File Name:  clean.go
//
// Author:  Jonathan Kans
//
// ==========================================================================

package eutils

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

var (
	externRunes    map[rune]string
	extRunesLoaded bool
)

// reencodes < and > to &lt and &gt, and & to $amp
var rfix *strings.Replacer

// Descriptions in comments taken from https://www.ssec.wisc.edu/~tomw/java/unicode.html

// runes are shown in Unicode Hexadecimal format

var extraRunes = map[rune]string{
	0x05BE: "-",           // HEBREW PUNCTUATION MAQAF
	0x115F: "(?)",         // HANGUL CHOSEONG FILLER
	0x25AF: "(rectangle)", // WHITE VERTICAL RECTANGLE
	0x25CF: "(circle)",    // BLACK CIRCLE
}

var germanCapitals = map[rune]string{
	0x0028: "(",  // LEFT PARENTHESIS
	0x0029: ")",  // RIGHT PARENTHESIS
	0x002D: "-",  // HYPHEN-MINUS
	0x0041: "A",  // LATIN CAPITAL LETTER A
	0x0042: "B",  // LATIN CAPITAL LETTER B
	0x0043: "C",  // LATIN CAPITAL LETTER C
	0x0044: "D",  // LATIN CAPITAL LETTER D
	0x0045: "E",  // LATIN CAPITAL LETTER E
	0x0046: "F",  // LATIN CAPITAL LETTER F
	0x0047: "G",  // LATIN CAPITAL LETTER G
	0x0048: "H",  // LATIN CAPITAL LETTER H
	0x0049: "I",  // LATIN CAPITAL LETTER I
	0x004A: "J",  // LATIN CAPITAL LETTER J
	0x004B: "K",  // LATIN CAPITAL LETTER K
	0x004C: "L",  // LATIN CAPITAL LETTER L
	0x004D: "M",  // LATIN CAPITAL LETTER M
	0x004E: "N",  // LATIN CAPITAL LETTER N
	0x004F: "O",  // LATIN CAPITAL LETTER O
	0x0050: "P",  // LATIN CAPITAL LETTER P
	0x0051: "Q",  // LATIN CAPITAL LETTER Q
	0x0052: "R",  // LATIN CAPITAL LETTER R
	0x0053: "S",  // LATIN CAPITAL LETTER S
	0x0054: "T",  // LATIN CAPITAL LETTER T
	0x0055: "U",  // LATIN CAPITAL LETTER U
	0x0056: "V",  // LATIN CAPITAL LETTER V
	0x0057: "W",  // LATIN CAPITAL LETTER W
	0x0058: "X",  // LATIN CAPITAL LETTER X
	0x0059: "Y",  // LATIN CAPITAL LETTER Y
	0x005A: "Z",  // LATIN CAPITAL LETTER Z
	0x00C4: "A",  // LATIN CAPITAL LETTER A WITH DIAERESIS
	0x00D6: "O",  // LATIN CAPITAL LETTER O WITH DIAERESIS
	0x00DC: "U",  // LATIN CAPITAL LETTER U WITH DIAERESIS
	0x00DF: "ss", // LATIN SMALL LETTER SHARP S
	0x1E9E: "SS", //
}

var germanRunes = map[rune]string{
	0x0028: "(",  // LEFT PARENTHESIS
	0x0029: ")",  // RIGHT PARENTHESIS
	0x002D: "-",  // HYPHEN-MINUS
	0x0041: "A",  // LATIN CAPITAL LETTER A
	0x0042: "B",  // LATIN CAPITAL LETTER B
	0x0043: "C",  // LATIN CAPITAL LETTER C
	0x0044: "D",  // LATIN CAPITAL LETTER D
	0x0045: "E",  // LATIN CAPITAL LETTER E
	0x0046: "F",  // LATIN CAPITAL LETTER F
	0x0047: "G",  // LATIN CAPITAL LETTER G
	0x0048: "H",  // LATIN CAPITAL LETTER H
	0x0049: "I",  // LATIN CAPITAL LETTER I
	0x004A: "J",  // LATIN CAPITAL LETTER J
	0x004B: "K",  // LATIN CAPITAL LETTER K
	0x004C: "L",  // LATIN CAPITAL LETTER L
	0x004D: "M",  // LATIN CAPITAL LETTER M
	0x004E: "N",  // LATIN CAPITAL LETTER N
	0x004F: "O",  // LATIN CAPITAL LETTER O
	0x0050: "P",  // LATIN CAPITAL LETTER P
	0x0051: "Q",  // LATIN CAPITAL LETTER Q
	0x0052: "R",  // LATIN CAPITAL LETTER R
	0x0053: "S",  // LATIN CAPITAL LETTER S
	0x0054: "T",  // LATIN CAPITAL LETTER T
	0x0055: "U",  // LATIN CAPITAL LETTER U
	0x0056: "V",  // LATIN CAPITAL LETTER V
	0x0057: "W",  // LATIN CAPITAL LETTER W
	0x0058: "X",  // LATIN CAPITAL LETTER X
	0x0059: "Y",  // LATIN CAPITAL LETTER Y
	0x005A: "Z",  // LATIN CAPITAL LETTER Z
	0x0061: "a",  // LATIN SMALL LETTER A
	0x0062: "b",  // LATIN SMALL LETTER B
	0x0063: "c",  // LATIN SMALL LETTER C
	0x0064: "d",  // LATIN SMALL LETTER D
	0x0065: "e",  // LATIN SMALL LETTER E
	0x0066: "f",  // LATIN SMALL LETTER F
	0x0067: "g",  // LATIN SMALL LETTER G
	0x0068: "h",  // LATIN SMALL LETTER H
	0x0069: "i",  // LATIN SMALL LETTER I
	0x006A: "j",  // LATIN SMALL LETTER J
	0x006B: "k",  // LATIN SMALL LETTER K
	0x006C: "l",  // LATIN SMALL LETTER L
	0x006D: "m",  // LATIN SMALL LETTER M
	0x006E: "n",  // LATIN SMALL LETTER N
	0x006F: "o",  // LATIN SMALL LETTER O
	0x0070: "p",  // LATIN SMALL LETTER P
	0x0071: "q",  // LATIN SMALL LETTER Q
	0x0072: "r",  // LATIN SMALL LETTER R
	0x0073: "s",  // LATIN SMALL LETTER S
	0x0074: "t",  // LATIN SMALL LETTER T
	0x0075: "u",  // LATIN SMALL LETTER U
	0x0076: "v",  // LATIN SMALL LETTER V
	0x0077: "w",  // LATIN SMALL LETTER W
	0x0078: "x",  // LATIN SMALL LETTER X
	0x0079: "y",  // LATIN SMALL LETTER Y
	0x007A: "z",  // LATIN SMALL LETTER Z
	0x00DF: "ss", // LATIN SMALL LETTER SHARP S
	0x00E4: "a",  // LATIN SMALL LETTER A WITH DIAERESIS
	0x00F6: "o",  // LATIN SMALL LETTER O WITH DIAERESIS
	0x00FC: "u",  // LATIN SMALL LETTER U WITH DIAERESIS
	0x00C4: "A",  // LATIN CAPITAL LETTER A WITH DIAERESIS
	0x00D6: "O",  // LATIN CAPITAL LETTER O WITH DIAERESIS
	0x00DC: "U",  // LATIN CAPITAL LETTER U WITH DIAERESIS
	0x1E9E: "SS", //
}

var germanVowels = map[rune]string{
	0x0041: "A", // LATIN CAPITAL LETTER A
	0x0045: "E", // LATIN CAPITAL LETTER E
	0x0049: "I", // LATIN CAPITAL LETTER I
	0x004F: "O", // LATIN CAPITAL LETTER O
	0x0055: "U", // LATIN CAPITAL LETTER U
	0x0061: "a", // LATIN SMALL LETTER A
	0x0065: "e", // LATIN SMALL LETTER E
	0x0069: "i", // LATIN SMALL LETTER I
	0x006F: "o", // LATIN SMALL LETTER O
	0x0075: "u", // LATIN SMALL LETTER U
	0x00E4: "a", // LATIN SMALL LETTER A WITH DIAERESIS
	0x00F6: "o", // LATIN SMALL LETTER O WITH DIAERESIS
	0x00FC: "u", // LATIN SMALL LETTER U WITH DIAERESIS
	0x00C4: "A", // LATIN CAPITAL LETTER A WITH DIAERESIS
	0x00D6: "O", // LATIN CAPITAL LETTER O WITH DIAERESIS
	0x00DC: "U", // LATIN CAPITAL LETTER U WITH DIAERESIS
}

var greekRunes = map[rune]string{
	0x0190: "epsilon", // LATIN CAPITAL LETTER OPEN E
	0x025B: "epsilon", // LATIN SMALL LETTER OPEN E
	0x0391: "alpha",   // GREEK CAPITAL LETTER ALPHA
	0x0392: "beta",    // GREEK CAPITAL LETTER BETA
	0x0393: "gamma",   // GREEK CAPITAL LETTER GAMMA
	0x0394: "delta",   // GREEK CAPITAL LETTER DELTA
	0x0395: "epsilon", // GREEK CAPITAL LETTER EPSILON
	0x0396: "zeta",    // GREEK CAPITAL LETTER ZETA
	0x0397: "eta",     // GREEK CAPITAL LETTER ETA
	0x0398: "theta",   // GREEK CAPITAL LETTER THETA
	0x0399: "iota",    // GREEK CAPITAL LETTER IOTA
	0x039A: "kappa",   // GREEK CAPITAL LETTER KAPPA
	0x039B: "lambda",  // GREEK CAPITAL LETTER LAMDA
	0x039C: "mu",      // GREEK CAPITAL LETTER MU
	0x039D: "nu",      // GREEK CAPITAL LETTER NU
	0x039E: "xi",      // GREEK CAPITAL LETTER XI
	0x039F: "omicron", // GREEK CAPITAL LETTER OMICRON
	0x03A0: "pi",      // GREEK CAPITAL LETTER PI
	0x03A1: "rho",     // GREEK CAPITAL LETTER RHO
	0x03A3: "sigma",   // GREEK CAPITAL LETTER SIGMA
	0x03A4: "tau",     // GREEK CAPITAL LETTER TAU
	0x03A5: "upsilon", // GREEK CAPITAL LETTER UPSILON
	0x03A6: "phi",     // GREEK CAPITAL LETTER PHI
	0x03A7: "chi",     // GREEK CAPITAL LETTER CHI
	0x03A8: "psi",     // GREEK CAPITAL LETTER PSI
	0x03A9: "omega",   // GREEK CAPITAL LETTER OMEGA
	0x03B1: "alpha",   // GREEK SMALL LETTER ALPHA
	0x03B2: "beta",    // GREEK SMALL LETTER BETA
	0x03B3: "gamma",   // GREEK SMALL LETTER GAMMA
	0x03B4: "delta",   // GREEK SMALL LETTER DELTA
	0x03B5: "epsilon", // GREEK SMALL LETTER EPSILON
	0x03B6: "zeta",    // GREEK SMALL LETTER ZETA
	0x03B7: "eta",     // GREEK SMALL LETTER ETA
	0x03B8: "theta",   // GREEK SMALL LETTER THETA
	0x03B9: "iota",    // GREEK SMALL LETTER IOTA
	0x03BA: "kappa",   // GREEK SMALL LETTER KAPPA
	0x03BB: "lambda",  // GREEK SMALL LETTER LAMDA
	0x03BC: "mu",      // GREEK SMALL LETTER MU
	0x03BD: "nu",      // GREEK SMALL LETTER NU
	0x03BE: "xi",      // GREEK SMALL LETTER XI
	0x03BF: "omicron", // GREEK SMALL LETTER OMICRON
	0x03C0: "pi",      // GREEK SMALL LETTER PI
	0x03C1: "rho",     // GREEK SMALL LETTER RHO
	0x03C2: "sigma",   // GREEK SMALL LETTER FINAL SIGMA
	0x03C3: "sigma",   // GREEK SMALL LETTER SIGMA
	0x03C4: "tau",     // GREEK SMALL LETTER TAU
	0x03C5: "upsilon", // GREEK SMALL LETTER UPSILON
	0x03C6: "phi",     // GREEK SMALL LETTER PHI
	0x03C7: "chi",     // GREEK SMALL LETTER CHI
	0x03C8: "psi",     // GREEK SMALL LETTER PSI
	0x03C9: "omega",   // GREEK SMALL LETTER OMEGA
	0x03D0: "beta",    // GREEK BETA SYMBOL
	0x03D1: "theta",   // GREEK THETA SYMBOL
	0x03D5: "phi",     // GREEK PHI SYMBOL
	0x03D6: "pi",      // GREEK PI SYMBOL
	0x03F0: "kappa",   // GREEK KAPPA SYMBOL
	0x03F1: "rho",     // GREEK RHO SYMBOL
	0x03F5: "epsilon", //
	0x1D5D: "beta",    //
	0x1D66: "beta",    //
}

var symbolRunes = map[rune]string{
	0x20A0: "ECU",                       // EURO-CURRENCY SIGN
	0x20A1: "CL",                        // COLON SIGN
	0x20A2: "Cr",                        // CRUZEIRO SIGN
	0x20A3: "FF",                        // FRENCH FRANC SIGN
	0x20A4: "L",                         // LIRA SIGN
	0x20A5: "mil",                       // MILL SIGN
	0x20A6: "N",                         // NAIRA SIGN
	0x20A7: "Pts",                       // PESETA SIGN
	0x20A8: "Rs",                        // RUPEE SIGN
	0x20A9: "W",                         // WON SIGN
	0x20AA: "NS",                        // NEW SHEQEL SIGN
	0x20AB: "D",                         // DONG SIGN
	0x20AC: "EU",                        // EURO SIGN
	0x20AD: "K",                         // KIP SIGN
	0x20AE: "T",                         // TUGRIK SIGN
	0x20AF: "Dr",                        // DRACHMA SIGN
	0x20DB: "...",                       // COMBINING THREE DOTS ABOVE
	0x20DC: "....",                      // COMBINING FOUR DOTS ABOVE
	0x2102: " (Copf) ",                  // DOUBLE-STRUCK CAPITAL C
	0x2103: "degrees C",                 // DEGREE CELSIUS
	0x2105: " (incare) ",                // CARE OF
	0x2107: " (euler) ",                 // EULER CONSTANT
	0x2109: "degrees F",                 // DEGREE FAHRENHEIT
	0x210A: " (gscr) ",                  // SCRIPT SMALL G
	0x210B: " (hamilt) ",                // SCRIPT CAPITAL H
	0x210C: " (Hfr) ",                   // BLACK-LETTER CAPITAL H
	0x210D: " (Hopf) ",                  // DOUBLE-STRUCK CAPITAL H
	0x210E: " (planckh) ",               // PLANCK CONSTANT
	0x210F: " (planck) ",                // PLANCK CONSTANT OVER TWO PI
	0x2110: " (Iscr) ",                  // SCRIPT CAPITAL I
	0x2111: " (image) ",                 // BLACK-LETTER CAPITAL I
	0x2112: " (Lscr) ",                  // SCRIPT CAPITAL L
	0x2113: " (ell) ",                   // SCRIPT SMALL L
	0x2115: " (Nopf) ",                  // DOUBLE-STRUCK CAPITAL N
	0x2116: " (numero) ",                // NUMERO SIGN
	0x2117: " (copysr) ",                // SOUND RECORDING COPYRIGHT
	0x2118: " (weierp) ",                // SCRIPT CAPITAL P
	0x2119: " (Popf) ",                  // DOUBLE-STRUCK CAPITAL P
	0x211A: " (Qopf) ",                  // DOUBLE-STRUCK CAPITAL Q
	0x211B: " (Rscr) ",                  // SCRIPT CAPITAL R
	0x211C: " (real) ",                  // BLACK-LETTER CAPITAL R
	0x211D: " (Ropf) ",                  // DOUBLE-STRUCK CAPITAL R
	0x211E: " (rx) ",                    // PRESCRIPTION TAKE
	0x2120: " (sm) ",                    // SERVICE MARK
	0x2121: " (TEL) ",                   // TELEPHONE SIGN
	0x2122: " (trade) ",                 // TRADE MARK SIGN
	0x2124: " (Zopf) ",                  // DOUBLE-STRUCK CAPITAL Z
	0x2126: " (ohm) ",                   // OHM SIGN
	0x2127: " (mho) ",                   // INVERTED OHM SIGN
	0x2128: " (Zfr) ",                   // BLACK-LETTER CAPITAL Z
	0x2129: " (iiota) ",                 // TURNED GREEK SMALL LETTER IOTA
	0x212A: " (kelvin) ",                // KELVIN SIGN
	0x212B: "A",                         // ANGSTROM SIGN
	0x212C: " (bernou) ",                // SCRIPT CAPITAL B
	0x212D: "C",                         // BLACK-LETTER CAPITAL C
	0x212E: " (estimated) ",             // ESTIMATED SYMBOL
	0x212F: "e",                         // SCRIPT SMALL E
	0x2130: "E",                         // SCRIPT CAPITAL E
	0x2131: "F",                         // SCRIPT CAPITAL F
	0x2132: "F",                         // TURNED CAPITAL F
	0x2133: "M",                         // SCRIPT CAPITAL M
	0x2134: "o",                         // SCRIPT SMALL O
	0x2135: " (alef) ",                  // ALEF SYMBOL
	0x2136: " (beth) ",                  // BET SYMBOL
	0x2137: " (gimel) ",                 // GIMEL SYMBOL
	0x2138: " (daleth) ",                // DALET SYMBOL
	0x213B: " (FAX) ",                   //
	0x2145: "D",                         //
	0x2146: "d",                         //
	0x2147: "e",                         //
	0x2148: "i",                         //
	0x2149: "j",                         //
	0x214E: "F",                         //
	0x2150: " (1/7) ",                   //
	0x2151: " (1/9) ",                   //
	0x2152: " (1/10) ",                  //
	0x2153: " (1/3) ",                   // VULGAR FRACTION ONE THIRD
	0x2154: " (2/3) ",                   // VULGAR FRACTION TWO THIRDS
	0x2155: " (1/5) ",                   // VULGAR FRACTION ONE FIFTH
	0x2156: " (2/5) ",                   // VULGAR FRACTION TWO FIFTHS
	0x2157: " (3/5) ",                   // VULGAR FRACTION THREE FIFTHS
	0x2158: " (4/5) ",                   // VULGAR FRACTION FOUR FIFTHS
	0x2159: " (1/6) ",                   // VULGAR FRACTION ONE SIXTH
	0x215A: " (5/6) ",                   // VULGAR FRACTION FIVE SIXTHS
	0x215B: " (1/8) ",                   // VULGAR FRACTION ONE EIGHTH
	0x215C: " (3/8) ",                   // VULGAR FRACTION THREE EIGHTHS
	0x215D: " (5/8) ",                   // VULGAR FRACTION FIVE EIGHTHS
	0x215E: " (7/8) ",                   // VULGAR FRACTION SEVEN EIGHTHS
	0x215F: " 1/",                       // FRACTION NUMERATOR ONE
	0x2160: " I ",                       // ROMAN NUMERAL ONE
	0x2161: " II ",                      // ROMAN NUMERAL TWO
	0x2162: " III ",                     // ROMAN NUMERAL THREE
	0x2163: " IV ",                      // ROMAN NUMERAL FOUR
	0x2164: " V ",                       // ROMAN NUMERAL FIVE
	0x2165: " VI ",                      // ROMAN NUMERAL SIX
	0x2166: " VII ",                     // ROMAN NUMERAL SEVEN
	0x2167: " VIII ",                    // ROMAN NUMERAL EIGHT
	0x2168: " IX ",                      // ROMAN NUMERAL NINE
	0x2169: " X ",                       // ROMAN NUMERAL TEN
	0x216A: " XI ",                      // ROMAN NUMERAL ELEVEN
	0x216B: " XII ",                     // ROMAN NUMERAL TWELVE
	0x216C: " L ",                       // ROMAN NUMERAL FIFTY
	0x216D: " C ",                       // ROMAN NUMERAL ONE HUNDRED
	0x216E: " D ",                       // ROMAN NUMERAL FIVE HUNDRED
	0x216F: " M ",                       // ROMAN NUMERAL ONE THOUSAND
	0x2170: " i ",                       // SMALL ROMAN NUMERAL ONE
	0x2171: " ii ",                      // SMALL ROMAN NUMERAL TWO
	0x2172: " iii ",                     // SMALL ROMAN NUMERAL THREE
	0x2173: " iv ",                      // SMALL ROMAN NUMERAL FOUR
	0x2174: " v ",                       // SMALL ROMAN NUMERAL FIVE
	0x2175: " vi ",                      // SMALL ROMAN NUMERAL SIX
	0x2176: " vii ",                     // SMALL ROMAN NUMERAL SEVEN
	0x2177: " viii ",                    // SMALL ROMAN NUMERAL EIGHT
	0x2178: " ix ",                      // SMALL ROMAN NUMERAL NINE
	0x2179: " x ",                       // SMALL ROMAN NUMERAL TEN
	0x217A: " xi ",                      // SMALL ROMAN NUMERAL ELEVEN
	0x217B: " xii ",                     // SMALL ROMAN NUMERAL TWELVE
	0x217C: " l ",                       // SMALL ROMAN NUMERAL FIFTY
	0x217D: " c ",                       // SMALL ROMAN NUMERAL ONE HUNDRED
	0x217E: " d ",                       // SMALL ROMAN NUMERAL FIVE HUNDRED
	0x217F: " m ",                       // SMALL ROMAN NUMERAL ONE THOUSAND
	0x2180: "(D",                        // ROMAN NUMERAL ONE THOUSAND C D
	0x2181: "D)",                        // ROMAN NUMERAL FIVE THOUSAND
	0x2182: "((|))",                     // ROMAN NUMERAL TEN THOUSAND
	0x2183: ")",                         // ROMAN NUMERAL REVERSED ONE HUNDRED
	0x2190: "-",                         // LEFTWARDS ARROW
	0x2191: "|",                         // UPWARDS ARROW
	0x2192: "-",                         // RIGHTWARDS ARROW
	0x2193: "|",                         // DOWNWARDS ARROW
	0x2194: "-",                         // LEFT RIGHT ARROW
	0x2195: "|",                         // UP DOWN ARROW
	0x2196: "\\",                        // NORTH WEST ARROW
	0x2197: "/",                         // NORTH EAST ARROW
	0x2198: "\\",                        // SOUTH EAST ARROW
	0x2199: "/",                         // SOUTH WEST ARROW
	0x219A: "-",                         // LEFTWARDS ARROW WITH STROKE
	0x219B: "-",                         // RIGHTWARDS ARROW WITH STROKE
	0x219C: "~",                         // LEFTWARDS WAVE ARROW
	0x219D: "~",                         // RIGHTWARDS WAVE ARROW
	0x219E: "-",                         // LEFTWARDS TWO HEADED ARROW
	0x219F: "|",                         // UPWARDS TWO HEADED ARROW
	0x21A0: "-",                         // RIGHTWARDS TWO HEADED ARROW
	0x21A1: "|",                         // DOWNWARDS TWO HEADED ARROW
	0x21A2: "-",                         // LEFTWARDS ARROW WITH TAIL
	0x21A3: "-",                         // RIGHTWARDS ARROW WITH TAIL
	0x21A4: "-",                         // LEFTWARDS ARROW FROM BAR
	0x21A5: "|",                         // UPWARDS ARROW FROM BAR
	0x21A6: "-",                         // RIGHTWARDS ARROW FROM BAR
	0x21A7: "|",                         // DOWNWARDS ARROW FROM BAR
	0x21A8: "|",                         // UP DOWN ARROW WITH BASE
	0x21A9: "-",                         // LEFTWARDS ARROW WITH HOOK
	0x21AA: "-",                         // RIGHTWARDS ARROW WITH HOOK
	0x21AB: "-",                         // LEFTWARDS ARROW WITH LOOP
	0x21AC: "-",                         // RIGHTWARDS ARROW WITH LOOP
	0x21AD: "-",                         // LEFT RIGHT WAVE ARROW
	0x21AE: "-",                         // LEFT RIGHT ARROW WITH STROKE
	0x21AF: "|",                         // DOWNWARDS ZIGZAG ARROW
	0x21B0: "|",                         // UPWARDS ARROW WITH TIP LEFTWARDS
	0x21B1: "|",                         // UPWARDS ARROW WITH TIP RIGHTWARDS
	0x21B2: "|",                         // DOWNWARDS ARROW WITH TIP LEFTWARDS
	0x21B3: "|",                         // DOWNWARDS ARROW WITH TIP RIGHTWARDS
	0x21B4: "|",                         // RIGHTWARDS ARROW WITH CORNER DOWNWARDS
	0x21B5: "|",                         // DOWNWARDS ARROW WITH CORNER LEFTWARDS
	0x21B6: "^",                         // ANTICLOCKWISE TOP SEMICIRCLE ARROW
	0x21B7: "V",                         // CLOCKWISE TOP SEMICIRCLE ARROW
	0x21B8: "\\",                        // NORTH WEST ARROW TO LONG BAR
	0x21B9: "=",                         // LEFTWARDS ARROW TO BAR OVER RIGHTWARDS ARROW TO BAR
	0x21BA: " (olarr) ",                 // ANTICLOCKWISE OPEN CIRCLE ARROW
	0x21BB: " (orarr) ",                 // CLOCKWISE OPEN CIRCLE ARROW
	0x21BC: " (lharu) ",                 // LEFTWARDS HARPOON WITH BARB UPWARDS
	0x21BD: " (lhard) ",                 // LEFTWARDS HARPOON WITH BARB DOWNWARDS
	0x21BE: " (uharr) ",                 // UPWARDS HARPOON WITH BARB RIGHTWARDS
	0x21BF: " (uharl) ",                 // UPWARDS HARPOON WITH BARB LEFTWARDS
	0x21C0: " (rharu) ",                 // RIGHTWARDS HARPOON WITH BARB UPWAR
	0x21C1: " (rhard) ",                 // RIGHTWARDS HARPOON WITH BARB DOWNW
	0x21C2: " (dharr) ",                 // DOWNWARDS HARPOON WITH BARB RIGHTW
	0x21C3: " (dharl) ",                 // DOWNWARDS HARPOON WITH BARB LEFTWA
	0x21C4: " (rlarr) ",                 // RIGHTWARDS ARROW OVER LEFTWARDS AR
	0x21C5: " (udarr) ",                 // UPWARDS ARROW LEFTWARDS OF DOWNWAR
	0x21C6: " (lrarr) ",                 // LEFTWARDS ARROW OVER RIGHTWARDS AR
	0x21C7: " (llarr) ",                 // LEFTWARDS PAIRED ARROWS
	0x21C8: " (uuarr) ",                 // UPWARDS PAIRED ARROWS
	0x21C9: " (rrarr) ",                 // RIGHTWARDS PAIRED ARROWS
	0x21CA: " (ddarr) ",                 // DOWNWARDS PAIRED ARROWS
	0x21CB: " (lrhar) ",                 // LEFTWARDS HARPOON OVER RIGHTWARDS
	0x21CC: " (rlhar) ",                 // RIGHTWARDS HARPOON OVER LEFTWARDS
	0x21CD: " (nlArr) ",                 // LEFTWARDS DOUBLE ARROW WITH STROKE
	0x21CE: " (nhArr) ",                 // LEFT RIGHT DOUBLE ARROW WITH STROK
	0x21CF: " (nrArr) ",                 // RIGHTWARDS DOUBLE ARROW WITH STROK
	0x21D0: " (lArr) ",                  // LEFTWARDS DOUBLE ARROW
	0x21D1: " (uArr) ",                  // UPWARDS DOUBLE ARROW
	0x21D2: " (rArr) ",                  // RIGHTWARDS DOUBLE ARROW
	0x21D3: " (dArr) ",                  // DOWNWARDS DOUBLE ARROW
	0x21D4: " (hArr) ",                  // LEFT RIGHT DOUBLE ARROW
	0x21D5: " (vArr) ",                  // UP DOWN DOUBLE ARROW
	0x21D6: " (nwArr) ",                 // NORTH WEST DOUBLE ARROW
	0x21D7: " (neArr) ",                 // NORTH EAST DOUBLE ARROW
	0x21D8: " (seArr) ",                 // SOUTH EAST DOUBLE ARROW
	0x21D9: " (swArr) ",                 // SOUTH WEST DOUBLE ARROW
	0x21DA: " (lAarr) ",                 // LEFTWARDS TRIPLE ARROW
	0x21DB: " (rAarr) ",                 // RIGHTWARDS TRIPLE ARROW
	0x21DC: " (ziglarr) ",               // LEFTWARDS SQUIGGLE ARROW
	0x21DD: " (zigrarr) ",               // RIGHTWARDS SQUIGGLE ARROW
	0x21DE: "|",                         // UPWARDS ARROW WITH DOUBLE STROKE
	0x21DF: "|",                         // DOWNWARDS ARROW WITH DOUBLE STROKE
	0x21E0: "-",                         // LEFTWARDS DASHED ARROW
	0x21E1: "|",                         // UPWARDS DASHED ARROW
	0x21E2: "-",                         // RIGHTWARDS DASHED ARROW
	0x21E3: "|",                         // DOWNWARDS DASHED ARROW
	0x21E4: " (larrb) ",                 // LEFTWARDS ARROW TO BAR
	0x21E5: " (rarrb) ",                 // RIGHTWARDS ARROW TO BAR
	0x21E6: "-",                         // LEFTWARDS WHITE ARROW
	0x21E7: "|",                         // UPWARDS WHITE ARROW
	0x21E8: "-",                         // RIGHTWARDS WHITE ARROW
	0x21E9: "|",                         // DOWNWARDS WHITE ARROW
	0x21EA: "|",                         // UPWARDS WHITE ARROW FROM BAR
	0x21EB: "|",                         // UPWARDS WHITE ARROW ON PEDESTAL
	0x21EC: "|",                         // UPWARDS WHITE ARROW ON PEDESTAL WITH HORIZONTAL BAR
	0x21ED: "|",                         // UPWARDS WHITE ARROW ON PEDESTAL WITH VERTICAL BAR
	0x21EE: "|",                         // UPWARDS WHITE DOUBLE ARROW
	0x21EF: "|",                         // UPWARDS WHITE DOUBLE ARROW ON PEDESTAL
	0x21F0: "-",                         // RIGHTWARDS WHITE ARROW FROM WALL
	0x21F1: "\\",                        // NORTH WEST ARROW TO CORNER
	0x21F2: "\\",                        // SOUTH EAST ARROW TO CORNER
	0x21F3: "|",                         // UP DOWN WHITE ARROW
	0x21FF: " (hoarr)",                  //
	0x2200: " (forall)",                 // FOR ALL
	0x2201: " (comp)) ",                 // COMPLEMENT
	0x2202: " (part) ",                  // PARTIAL DIFFERENTIAL
	0x2203: " (exist)) ",                // THERE EXISTS
	0x2204: " (nexist) ",                // THERE DOES NOT EXIST
	0x2205: " (empty) ",                 // EMPTY SET
	0x2206: " (increment) ",             // INCREMENT
	0x2207: " (nabla) ",                 // NABLA
	0x2208: " (isin) ",                  // ELEMENT OF
	0x2209: " (notin) ",                 // NOT AN ELEMENT OF
	0x220B: " (ni)",                     // CONTAINS AS MEMBER
	0x220C: " (notni) ",                 // DOES NOT CONTAIN AS MEMBER
	0x220F: " (product) ",               // N-ARY PRODUCT
	0x2210: " (coprod) ",                // N-ARY COPRODUCT
	0x2211: " (sum) ",                   // N-ARY SUMMATION
	0x2212: " (minus) ",                 // MINUS SIGN
	0x2213: " (mnplus) ",                // MINUS-OR-PLUS SIGN
	0x2214: " (plusdo) ",                // DOT PLUS
	0x2215: " (division slash) ",        // DIVISION SLASH
	0x2216: " (setminus) ",              // SET MINUS
	0x2217: " (lowast) ",                // ASTERISK OPERATOR
	0x2218: " (compfn) ",                // RING OPERATOR
	0x2219: " (bullet) ",                // BULLET OPERATOR
	0x221A: " (radic) ",                 // SQUARE ROOT
	0x221B: " (cube root) ",             // CUBE ROOT
	0x221C: " (fourth root) ",           // FOURTH ROOT
	0x221D: " (prop) ",                  // PROPORTIONAL TO
	0x221E: " (infin) ",                 // INFINITY
	0x221F: " (angrt) ",                 // RIGHT ANGLE
	0x2220: " (ang) ",                   // ANGLE
	0x2221: " (angmsd) ",                // MEASURED ANGLE
	0x2222: " (angsph) ",                // SPHERICAL ANGLE
	0x2223: " (mid) ",                   // DIVIDES
	0x2224: " (nmid) ",                  // DOES NOT DIVIDE
	0x2225: " (parallel) ",              // PARALLEL TO
	0x2226: " (npar) ",                  // NOT PARALLEL TO
	0x2227: " (and) ",                   // LOGICAL AND
	0x2228: " (or) ",                    // LOGICAL OR
	0x2229: " (cap) ",                   // INTERSECTION
	0x222A: " (cup) ",                   // UNION
	0x222B: " (int) ",                   // INTEGRAL
	0x222C: " (Int) ",                   // DOUBLE INTEGRAL
	0x222D: " (iiint) ",                 // TRIPLE INTEGRAL
	0x222E: " (conint) ",                // CONTOUR INTEGRAL
	0x222F: " (Conint) ",                // SURFACE INTEGRAL
	0x2230: " (Cconint) ",               // VOLUME INTEGRAL
	0x2231: " (cwint) ",                 // CLOCKWISE INTEGRAL
	0x2232: " (cwconint) ",              // CLOCKWISE CONTOUR INTEGRAL
	0x2233: " (awconint) ",              // ANTICLOCKWISE CONTOUR INTEGRAL
	0x2234: " (there4) ",                // THEREFORE
	0x2235: " (because) ",               // BECAUSE
	0x2236: " : ",                       // RATIO
	0x2237: " :: ",                      // PROPORTION
	0x2238: " (minusd) ",                // DOT MINUS
	0x223A: " (mDDot) ",                 // GEOMETRIC PROPORTION
	0x223B: " (homth) ",                 // HOMOTHETIC
	0x223C: " (sim) ",                   // TILDE OPERATOR
	0x223D: " (bsim) ",                  // REVERSED TILDE
	0x223E: " (ac) ",                    // INVERTED LAZY S
	0x223F: " (acd) ",                   // SINE WAVE
	0x2240: " (wreath) ",                // WREATH PRODUCT
	0x2241: " (nsim) ",                  // NOT TILDE
	0x2242: " (esim) ",                  // MINUS TILDE
	0x2243: " (sime) ",                  // ASYMPTOTICALLY EQUAL TO
	0x2244: " (nsime) ",                 // NOT ASYMPTOTICALLY EQUAL TO
	0x2245: " (cong) ",                  // APPROXIMATELY EQUAL TO
	0x2246: " (simne) ",                 // APPROXIMATELY BUT NOT ACTUALLY EQUAL TO
	0x2247: " (ncong) ",                 // NEITHER APPROXIMATELY NOR ACTUALLY EQUAL TO
	0x2248: " (asymp) ",                 // ALMOST EQUAL TO
	0x2249: " (nap) ",                   // NOT ALMOST EQUAL TO
	0x224A: " (approxeq)",               // ALMOST EQUAL OR EQUAL TO
	0x224B: " (apid) ",                  // TRIPLE TILDE
	0x224C: " (bcong) ",                 // ALL EQUAL TO
	0x224D: " (asympeq) ",               // EQUIVALENT TO
	0x224E: " (bump) ",                  // GEOMETRICALLY EQUIVALENT TO
	0x224F: " (bumpe) ",                 // DIFFERENCE BETWEEN
	0x2250: " (esdot) ",                 // APPROACHES THE LIMIT
	0x2251: " (eDot) ",                  // GEOMETRICALLY EQUAL TO
	0x2252: " (efDot) ",                 // APPROXIMATELY EQUAL TO OR THE IMAGE OF
	0x2253: " (erDot) ",                 // IMAGE OF OR APPROXIMATELY EQUAL TO
	0x2254: " (colone) ",                // COLON EQUALS
	0x2255: " (ecolon) ",                // EQUALS COLON
	0x2256: " (ecir) ",                  // RING IN EQUAL TO
	0x2257: " (cire) ",                  // RING EQUAL TO
	0x2259: " (wedgeq)",                 // ESTIMATES
	0x225A: " (veeeq) ",                 // EQUIANGULAR TO
	0x225B: " (star eq) ",               // STAR EQUALS
	0x225C: " (trie) ",                  // DELTA EQUAL TO
	0x225F: " (equest) ",                // QUESTIONED EQUAL TO
	0x2260: " (ne) ",                    // NOT EQUAL TO
	0x2261: " (equiv) ",                 // IDENTICAL TO
	0x2262: " (nequiv) ",                // NOT IDENTICAL TO
	0x2264: " <= ",                      // LESS-THAN OR EQUAL TO
	0x2265: " >= ",                      // GREATER-THAN OR EQUAL TO
	0x2266: " <= ",                      // LESS-THAN OVER EQUAL TO
	0x2267: " >= ",                      // GREATER-THAN OVER EQUAL TO
	0x2268: " (lnE) ",                   // LESS-THAN BUT NOT EQUAL TO
	0x2269: " (gnE) ",                   // GREATER-THAN BUT NOT EQUAL TO
	0x226A: " << ",                      // MUCH LESS-THAN
	0x226B: " >> ",                      // MUCH GREATER-THAN
	0x226C: " (between) ",               // BETWEEN
	0x226D: " (NotCupCap) ",             // NOT EQUIVALENT TO
	0x226E: " (nlt) ",                   // NOT LESS-THAN
	0x226F: " (ngt) ",                   // NOT GREATER-THAN
	0x2270: " (nle) ",                   // NEITHER LESS-THAN NOR EQUAL TO
	0x2271: " (nge) ",                   // NEITHER GREATER-THAN NOR EQUAL TO
	0x2272: " (lsim) ",                  // LESS-THAN OR EQUIVALENT TO
	0x2273: " (gsim) ",                  // GREATER-THAN OR EQUIVALENT TO
	0x2274: " (nlsim) ",                 // NEITHER LESS-THAN NOR EQUIVALENT TO
	0x2275: " (ngsim) ",                 // NEITHER GREATER-THAN NOR EQUIVALENT TO
	0x2276: " <> ",                      // LESS-THAN OR GREATER-THAN
	0x2277: " >< ",                      // GREATER-THAN OR LESS-THAN
	0x2278: " (ntlg) ",                  // NEITHER LESS-THAN NOR GREATER-THAN
	0x2279: " (ntgl) ",                  // NEITHER GREATER-THAN NOR LESS-THAN
	0x227A: " (pr) ",                    // PRECEDES
	0x227B: " (sc) ",                    // SUCCEEDS
	0x227C: " (prcue) ",                 // PRECEDES OR EQUAL TO
	0x227D: " (sccue) ",                 // SUCCEEDS OR EQUAL TO
	0x227E: " (prsim) ",                 // PRECEDES OR EQUIVALENT TO
	0x227F: " (scsim) ",                 // SUCCEEDS OR EQUIVALENT TO
	0x2280: " (npr) ",                   // DOES NOT PRECEDE
	0x2281: " (nsc) ",                   // DOES NOT SUCCEED
	0x2282: " (sub) ",                   // SUBSET OF
	0x2283: " (sup) ",                   // SUPERSET OF
	0x2284: " (nsub) ",                  // NOT A SUBSET OF
	0x2285: " (nsup) ",                  // NOT A SUPERSET OF
	0x2286: " (sube) ",                  // SUBSET OF OR EQUAL TO
	0x2287: " (supe) ",                  // SUPERSET OF OR EQUAL TO
	0x2288: " (nsube) ",                 // NEITHER A SUBSET OF NOR EQUAL TO
	0x2289: " (nsupe) ",                 // NEITHER A SUPERSET OF NOR EQUAL TO
	0x228A: " (subne) ",                 // SUBSET OF WITH NOT EQUAL TO
	0x228B: " (supne) ",                 // SUPERSET OF WITH NOT EQUAL TO
	0x228D: " (cupdot) ",                // MULTISET MULTIPLICATION
	0x228E: " (uplus) ",                 // MULTISET UNION
	0x228F: " (sqsub) ",                 // SQUARE IMAGE OF
	0x2290: " (sqsup) ",                 // SQUARE ORIGINAL OF
	0x2291: " (sqsube) ",                // SQUARE IMAGE OF OR EQUAL TO
	0x2292: " (sqsupe) ",                // SQUARE ORIGINAL OF OR EQUAL TO
	0x2293: " (sqcap) )",                // SQUARE CAP
	0x2294: " (sqcup) ",                 // SQUARE CUP
	0x2295: " (oplus) ",                 // CIRCLED PLUS
	0x2296: " (ominus) ",                // CIRCLED MINUS
	0x2297: " (ominus) ",                // CIRCLED TIMES
	0x2298: " (osol) ",                  // CIRCLED DIVISION SLASH
	0x2299: " (odot) ",                  // CIRCLED DOT OPERATOR
	0x229A: " (ocir) ",                  // CIRCLED RING OPERATOR
	0x229B: " (oast) ",                  // CIRCLED ASTERISK OPERATOR
	0x229D: " (odash) ",                 // CIRCLED DASH
	0x229E: " (plusb) ",                 // SQUARED PLUS
	0x229F: " (minusb) ",                // SQUARED MINUS
	0x22A0: " (timesb) ",                // SQUARED TIMES
	0x22A1: " (sdotb) ",                 // SQUARED DOT OPERATOR
	0x22A2: " (vdash) ",                 // RIGHT TACK
	0x22A3: " (dashv) ",                 // LEFT TACK
	0x22A4: " (top) ",                   // DOWN TACK
	0x22A5: " (perp) ",                  // UP TACK
	0x22A7: " (models) ",                // MODELS
	0x22A8: " (vDash) ",                 // TRUE
	0x22A9: " (Vdash) ",                 // FORCES
	0x22AA: " (Vvdash) ",                // TRIPLE VERTICAL BAR RIGHT TURNSTILE
	0x22AB: " (VDash) ",                 // DOUBLE VERTICAL BAR DOUBLE RIGHT TURNSTILE
	0x22AC: " (nvdash) ",                // DOES NOT PROVE
	0x22AD: " (nvDash) ",                // NOT TRUE
	0x22AE: " (nVdash) ",                // DOES NOT FORCE
	0x22AF: " (nVDash) ",                // NEGATED DOUBLE VERTICAL BAR DOUBLE RIGHT TURNSTILE
	0x22B0: " (prurel) ",                // PRECEDES UNDER RELATION
	0x22B2: " (vltri) ",                 // NORMAL SUBGROUP OF
	0x22B3: " (vrtri) ",                 // CONTAINS AS NORMAL SUBGROUP
	0x22B4: " (ltrie) ",                 // NORMAL SUBGROUP OF OR EQUAL TO
	0x22B5: " (rtrie) ",                 // CONTAINS AS NORMAL SUBGROUP OR EQUAL TO
	0x22B6: " (origof) ",                // ORIGINAL OF
	0x22B7: " (imof) ",                  // IMAGE OF
	0x22B8: " (mumap) ",                 // MULTIMAP
	0x22B9: " (hercon) ",                // HERMITIAN CONJUGATE MATRIX
	0x22BA: " (intcal) ",                // INTERCALATE
	0x22BB: " (veebar) ",                // XOR
	0x22BC: " (Nand) ",                  // NAND
	0x22BD: " (barvee) ",                // NOR
	0x22BE: " (angrtvb) ",               // RIGHT ANGLE WITH ARC
	0x22BF: " (lrtri) ",                 // RIGHT TRIANGLE
	0x22C0: " (xwedge) ",                // N-ARY LOGICAL AND
	0x22C1: " (xvee) ",                  // N-ARY LOGICAL OR
	0x22C2: " (xcap) ",                  // N-ARY INTERSECTION
	0x22C3: " (xcup) ",                  // N-ARY UNION
	0x22C4: " (diamond) ",               // DIAMOND OPERATOR
	0x22C5: " (sdot) ",                  // DOT OPERATOR
	0x22C6: " (Star) ",                  // STAR OPERATOR
	0x22C7: " (divonx) ",                // DIVISION TIMES
	0x22C8: " (bowtie) ",                // BOWTIE
	0x22C9: " (ltimes) ",                // LEFT NORMAL FACTOR SEMIDIRECT PRODUCT
	0x22CA: " (rtimes) ",                // RIGHT NORMAL FACTOR SEMIDIRECT PRODUCT
	0x22CB: " (lthree) ",                // LEFT SEMIDIRECT PRODUCT
	0x22CC: " (rthree) ",                // RIGHT SEMIDIRECT PRODUCT
	0x22CD: " (bsime) ",                 // REVERSED TILDE EQUALS
	0x22CE: " (cuvee) ",                 // CURLY LOGICAL OR
	0x22CF: " (cuwed) ",                 // CURLY LOGICAL AND
	0x22D0: " (Sub) ",                   // DOUBLE SUBSET
	0x22D1: " (Sup) ",                   // DOUBLE SUPERSET
	0x22D2: " (Cap) ",                   // DOUBLE INTERSECTION
	0x22D3: " (Cup) ",                   // DOUBLE UNION
	0x22D4: " (fork) ",                  // PITCHFORK
	0x22D5: " (epar) ",                  // EQUAL AND PARALLEL TO
	0x22D6: " (ltdot) ",                 // LESS-THAN WITH DOT
	0x22D7: " (gtdot) ",                 // GREATER-THAN WITH DOT
	0x22D8: " <<< ",                     // VERY MUCH LESS-THAN
	0x22D9: " >>> ",                     // VERY MUCH GREATER-THAN
	0x22DA: " <=> ",                     // LESS-THAN EQUAL TO OR GREATER-THAN
	0x22DB: " >=< ",                     // GREATER-THAN EQUAL TO OR LESS-THAN
	0x22DC: " =/< ",                     // EQUAL TO OR LESS-THAN
	0x22DD: " =/> ",                     // EQUAL TO OR GREATER-THAN
	0x22DE: " (cuepr) ",                 // EQUAL TO OR PRECEDES
	0x22DF: " (cuesc) ",                 // EQUAL TO OR SUCCEEDS
	0x22E0: " (nprcue) ",                // DOES NOT PRECEDE OR EQUAL
	0x22E1: " (nsccue) ",                // DOES NOT SUCCEED OR EQUAL
	0x22E2: " (nsqsube) ",               // NOT SQUARE IMAGE OF OR EQUAL TO
	0x22E3: " (nsqsupe) ",               // NOT SQUARE ORIGINAL OF OR EQUAL TO
	0x22E6: " (lnsim) ",                 // LESS-THAN BUT NOT EQUIVALENT TO
	0x22E7: " (gnsim)",                  // GREATER-THAN BUT NOT EQUIVALENT TO
	0x22E8: " (prnsim) ",                // PRECEDES BUT NOT EQUIVALENT TO
	0x22E9: " (scnsim) ",                // SUCCEEDS BUT NOT EQUIVALENT TO
	0x22EA: " (nltri) ",                 // NOT NORMAL SUBGROUP OF
	0x22EB: " (nrtri) ",                 // DOES NOT CONTAIN AS NORMAL SUBGROUP
	0x22EC: " (nltrie) ",                // NOT NORMAL SUBGROUP OF OR EQUAL TO
	0x22ED: " (nrtrie) ",                // DOES NOT CONTAIN AS NORMAL SUBGROUP OR EQUAL
	0x22EE: " (vellip) ",                // VERTICAL ELLIPSIS
	0x22EF: " (ctdot) ",                 // MIDLINE HORIZONTAL ELLIPSIS
	0x22F0: " (utdot) ",                 // UP RIGHT DIAGONAL ELLIPSIS
	0x22F1: " (dtdot) ",                 // DOWN RIGHT DIAGONAL ELLIPSIS
	0x2303: " ^ ",                       // UP ARROWHEAD
	0x2306: " (log dbl bar) ",           // PERSPECTIVE
	0x2308: " (lceil) ",                 // LEFT CEILING
	0x2309: " (rceil) ",                 // RIGHT CEILING
	0x230A: " (lfloor) ",                // LEFT FLOOR
	0x230B: " (rfloor) ",                // RIGHT FLOOR
	0x230C: "downward right crop mark ", // BOTTOM RIGHT CROP
	0x230D: "downward left crop mark ",  // BOTTOM LEFT CROP
	0x230E: "upward right crop mark ",   // TOP RIGHT CROP
	0x230F: "upward left crop mark ",    // TOP LEFT CROP
	0x2310: "reverse not",               // REVERSED NOT SIGN
	0x2312: "profile of a line",         // ARC
	0x2313: "profile of a surface",      // SEGMENT
	0x2315: "telephone recorder symbol", // TELEPHONE RECORDER
	0x2316: "register mark or target",   // POSITION INDICATOR
	0x231C: " upper left corner",        // TOP LEFT CORNER
	0x231D: " upper right corner",       // TOP RIGHT CORNER
	0x231E: " downward left corner",     // BOTTOM LEFT CORNER
	0x231F: " downward right corner",    // BOTTOM RIGHT CORNER
	0x2322: " down curve",               // FROWN
	0x2323: " up curve",                 // SMILE
	0x2329: " < ",                       // LEFT-POINTING ANGLE BRACKET
	0x232A: " > ",                       // RIGHT-POINTING ANGLE BRACKET
	0x232D: " (cylindricity) ",          // CYLINDRICITY
	0x232E: " (aa profile) ",            // ALL AROUND-PROFILE
	0x2336: " (I beam) ",                // APL FUNCTIONAL SYMBOL I-BEAM
	0x244A: "\\\\",                      // OCR DOUBLE BACKSLASH
	0x2460: " 1 ",                       // CIRCLED DIGIT ONE
	0x2461: " 2 ",                       // CIRCLED DIGIT TWO
	0x2462: " 3 ",                       // CIRCLED DIGIT THREE
	0x2463: " 4 ",                       // CIRCLED DIGIT FOUR
	0x2464: " 5 ",                       // CIRCLED DIGIT FIVE
	0x2465: " 6 ",                       // CIRCLED DIGIT SIX
	0x2466: " 7 ",                       // CIRCLED DIGIT SEVEN
	0x2467: " 8 ",                       // CIRCLED DIGIT EIGHT
	0x2468: " 9 ",                       // CIRCLED DIGIT NINE
	0x2469: " 10 ",                      // CIRCLED NUMBER TEN
	0x246A: " 11 ",                      // CIRCLED NUMBER ELEVEN
	0x246B: " 12 ",                      // CIRCLED NUMBER TWELVE
	0x246C: " 13 ",                      // CIRCLED NUMBER THIRTEEN
	0x246D: " 14 ",                      // CIRCLED NUMBER FOURTEEN
	0x246E: " 15 ",                      // CIRCLED NUMBER FIFTEEN
	0x246F: " 16 ",                      // CIRCLED NUMBER SIXTEEN
	0x2470: " 17 ",                      // CIRCLED NUMBER SEVENTEEN
	0x2471: " 18 ",                      // CIRCLED NUMBER EIGHTEEN
	0x2472: " 19 ",                      // CIRCLED NUMBER NINETEEN
	0x2473: " 20 ",                      // CIRCLED NUMBER TWENTY
	0x2474: " (1) ",                     // PARENTHESIZED DIGIT ONE
	0x2475: " (2) ",                     // PARENTHESIZED DIGIT TWO
	0x2476: " (3) ",                     // PARENTHESIZED DIGIT THREE
	0x2477: " (4) ",                     // PARENTHESIZED DIGIT FOUR
	0x2478: " (5) ",                     // PARENTHESIZED DIGIT FIVE
	0x2479: " (6) ",                     // PARENTHESIZED DIGIT SIX
	0x247A: " (7) ",                     // PARENTHESIZED DIGIT SEVEN
	0x247B: " (8) ",                     // PARENTHESIZED DIGIT EIGHT
	0x247C: " (9) ",                     // PARENTHESIZED DIGIT NINE
	0x247D: " (10) ",                    // PARENTHESIZED NUMBER TEN
	0x247E: " (11) ",                    // PARENTHESIZED NUMBER ELEVEN
	0x247F: " (12) ",                    // PARENTHESIZED NUMBER TWELVE
	0x2480: " (13) ",                    // PARENTHESIZED NUMBER THIRTEEN
	0x2481: " (14) ",                    // PARENTHESIZED NUMBER FOURTEEN
	0x2482: " (15) ",                    // PARENTHESIZED NUMBER FIFTEEN
	0x2483: " (16) ",                    // PARENTHESIZED NUMBER SIXTEEN
	0x2484: " (17) ",                    // PARENTHESIZED NUMBER SEVENTEEN
	0x2485: " (18) ",                    // PARENTHESIZED NUMBER EIGHTEEN
	0x2486: " (19) ",                    // PARENTHESIZED NUMBER NINETEEN
	0x2487: " (20) ",                    // PARENTHESIZED NUMBER TWENTY
	0x2488: " 1. ",                      // DIGIT ONE FULL STOP
	0x2489: " 2. ",                      // DIGIT TWO FULL STOP
	0x248A: " 3. ",                      // DIGIT THREE FULL STOP
	0x248B: " 4. ",                      // DIGIT FOUR FULL STOP
	0x248C: " 5. ",                      // DIGIT FIVE FULL STOP
	0x248D: " 6. ",                      // DIGIT SIX FULL STOP
	0x248E: " 7. ",                      // DIGIT SEVEN FULL STOP
	0x248F: " 8. ",                      // DIGIT EIGHT FULL STOP
	0x2490: " 9. ",                      // DIGIT NINE FULL STOP
	0x2491: " 10. ",                     // NUMBER TEN FULL STOP
	0x2492: " 11. ",                     // NUMBER ELEVEN FULL STOP
	0x2493: " 12. ",                     // NUMBER TWELVE FULL STOP
	0x2494: " 13. ",                     // NUMBER THIRTEEN FULL STOP
	0x2495: " 14. ",                     // NUMBER FOURTEEN FULL STOP
	0x2496: " 15. ",                     // NUMBER FIFTEEN FULL STOP
	0x2497: " 16. ",                     // NUMBER SIXTEEN FULL STOP
	0x2498: " 17. ",                     // NUMBER SEVENTEEN FULL STOP
	0x2499: " 18. ",                     // NUMBER EIGHTEEN FULL STOP
	0x249A: " 19. ",                     // NUMBER NINETEEN FULL STOP
	0x249B: " 20. ",                     // NUMBER TWENTY FULL STOP
	0x249C: " (a) ",                     // PARENTHESIZED LATIN SMALL LETTER A
	0x249D: " (b) ",                     // PARENTHESIZED LATIN SMALL LETTER B
	0x249E: " (c) ",                     // PARENTHESIZED LATIN SMALL LETTER C
	0x249F: " (d) ",                     // PARENTHESIZED LATIN SMALL LETTER D
	0x24A0: " (e) ",                     // PARENTHESIZED LATIN SMALL LETTER E
	0x24A1: " (f) ",                     // PARENTHESIZED LATIN SMALL LETTER F
	0x24A2: " (g) ",                     // PARENTHESIZED LATIN SMALL LETTER G
	0x24A3: " (h) ",                     // PARENTHESIZED LATIN SMALL LETTER H
	0x24A4: " (i) ",                     // PARENTHESIZED LATIN SMALL LETTER I
	0x24A5: " (j) ",                     // PARENTHESIZED LATIN SMALL LETTER J
	0x24A6: " (k) ",                     // PARENTHESIZED LATIN SMALL LETTER K
	0x24A7: " (l) ",                     // PARENTHESIZED LATIN SMALL LETTER L
	0x24A8: " (m) ",                     // PARENTHESIZED LATIN SMALL LETTER M
	0x24A9: " (n) ",                     // PARENTHESIZED LATIN SMALL LETTER N
	0x24AA: " (o) ",                     // PARENTHESIZED LATIN SMALL LETTER O
	0x24AB: " (p) ",                     // PARENTHESIZED LATIN SMALL LETTER P
	0x24AC: " (q) ",                     // PARENTHESIZED LATIN SMALL LETTER Q
	0x24AD: " (r) ",                     // PARENTHESIZED LATIN SMALL LETTER R
	0x24AE: " (s) ",                     // PARENTHESIZED LATIN SMALL LETTER S
	0x24AF: " (t) ",                     // PARENTHESIZED LATIN SMALL LETTER T
	0x24B0: " (u) ",                     // PARENTHESIZED LATIN SMALL LETTER U
	0x24B1: " (v) ",                     // PARENTHESIZED LATIN SMALL LETTER V
	0x24B2: " (w) ",                     // PARENTHESIZED LATIN SMALL LETTER W
	0x24B3: " (x) ",                     // PARENTHESIZED LATIN SMALL LETTER X
	0x24B4: " (y) ",                     // PARENTHESIZED LATIN SMALL LETTER Y
	0x24B5: " (z) ",                     // PARENTHESIZED LATIN SMALL LETTER Z
	0x24B6: " A ",                       // CIRCLED LATIN CAPITAL LETTER A
	0x24B7: " B ",                       // CIRCLED LATIN CAPITAL LETTER B
	0x24B8: " C ",                       // CIRCLED LATIN CAPITAL LETTER C
	0x24B9: " D ",                       // CIRCLED LATIN CAPITAL LETTER D
	0x24BA: " E ",                       // CIRCLED LATIN CAPITAL LETTER E
	0x24BB: " F ",                       // CIRCLED LATIN CAPITAL LETTER F
	0x24BC: " G ",                       // CIRCLED LATIN CAPITAL LETTER G
	0x24BD: " H ",                       // CIRCLED LATIN CAPITAL LETTER H
	0x24BE: " I ",                       // CIRCLED LATIN CAPITAL LETTER I
	0x24BF: " J ",                       // CIRCLED LATIN CAPITAL LETTER J
	0x24C0: " K ",                       // CIRCLED LATIN CAPITAL LETTER K
	0x24C1: " L ",                       // CIRCLED LATIN CAPITAL LETTER L
	0x24C2: " M ",                       // CIRCLED LATIN CAPITAL LETTER M
	0x24C3: " N ",                       // CIRCLED LATIN CAPITAL LETTER N
	0x24C4: " O ",                       // CIRCLED LATIN CAPITAL LETTER O
	0x24C5: " P ",                       // CIRCLED LATIN CAPITAL LETTER P
	0x24C6: " Q ",                       // CIRCLED LATIN CAPITAL LETTER Q
	0x24C7: " R ",                       // CIRCLED LATIN CAPITAL LETTER R
	0x24C8: " S ",                       // CIRCLED LATIN CAPITAL LETTER S
	0x24C9: " T ",                       // CIRCLED LATIN CAPITAL LETTER T
	0x24CA: " U ",                       // CIRCLED LATIN CAPITAL LETTER U
	0x24CB: " V ",                       // CIRCLED LATIN CAPITAL LETTER V
	0x24CC: " W ",                       // CIRCLED LATIN CAPITAL LETTER W
	0x24CD: " X ",                       // CIRCLED LATIN CAPITAL LETTER X
	0x24CE: " Y ",                       // CIRCLED LATIN CAPITAL LETTER Y
	0x24CF: " Z ",                       // CIRCLED LATIN CAPITAL LETTER Z
	0x24D0: " a ",                       // CIRCLED LATIN SMALL LETTER A
	0x24D1: " b ",                       // CIRCLED LATIN SMALL LETTER B
	0x24D2: " c ",                       // CIRCLED LATIN SMALL LETTER C
	0x24D3: " d ",                       // CIRCLED LATIN SMALL LETTER D
	0x24D4: " e ",                       // CIRCLED LATIN SMALL LETTER E
	0x24D5: " f ",                       // CIRCLED LATIN SMALL LETTER F
	0x24D6: " g ",                       // CIRCLED LATIN SMALL LETTER G
	0x24D7: " h ",                       // CIRCLED LATIN SMALL LETTER H
	0x24D8: " i ",                       // CIRCLED LATIN SMALL LETTER I
	0x24D9: " j ",                       // CIRCLED LATIN SMALL LETTER J
	0x24DA: " k ",                       // CIRCLED LATIN SMALL LETTER K
	0x24DB: " l ",                       // CIRCLED LATIN SMALL LETTER L
	0x24DC: " m ",                       // CIRCLED LATIN SMALL LETTER M
	0x24DD: " n ",                       // CIRCLED LATIN SMALL LETTER N
	0x24DE: " o ",                       // CIRCLED LATIN SMALL LETTER O
	0x24DF: " p ",                       // CIRCLED LATIN SMALL LETTER P
	0x24E0: " q ",                       // CIRCLED LATIN SMALL LETTER Q
	0x24E1: " r ",                       // CIRCLED LATIN SMALL LETTER R
	0x24E2: " s ",                       // CIRCLED LATIN SMALL LETTER S
	0x24E3: " t ",                       // CIRCLED LATIN SMALL LETTER T
	0x24E4: " u ",                       // CIRCLED LATIN SMALL LETTER U
	0x24E5: " v ",                       // CIRCLED LATIN SMALL LETTER V
	0x24E6: " w ",                       // CIRCLED LATIN SMALL LETTER W
	0x24E7: " x ",                       // CIRCLED LATIN SMALL LETTER X
	0x24E8: " y ",                       // CIRCLED LATIN SMALL LETTER Y
	0x24E9: " z ",                       // CIRCLED LATIN SMALL LETTER Z
	0x24EA: " 0 ",                       // CIRCLED DIGIT ZERO
	0x24EB: " 11 ",                      //
	0x24EC: " 12 ",                      //
	0x24ED: " 13 ",                      //
	0x24EE: " 14 ",                      //
	0x24EF: " 15 ",                      //
	0x24F0: " 16 ",                      //
	0x24F1: " 17 ",                      //
	0x24F2: " 18 ",                      //
	0x24F3: " 19 ",                      //
	0x24F4: " 20 ",                      //
	0x24F5: " 1 ",                       //
	0x24F6: " 2 ",                       //
	0x24F7: " 3 ",                       //
	0x24F8: " 4 ",                       //
	0x24F9: " 5 ",                       //
	0x24FA: " 6 ",                       //
	0x24FB: " 7 ",                       //
	0x24FC: " 8 ",                       //
	0x24FD: " 9 ",                       //
	0x24FE: " 10 ",                      //
	0x24FF: " 0 ",                       //
	0x2500: "-",                         // BOX DRAWINGS LIGHT HORIZONTAL
	0x2501: "-",                         // BOX DRAWINGS HEAVY HORIZONTAL
	0x2502: "|",                         // BOX DRAWINGS LIGHT VERTICAL
	0x2503: "|",                         // BOX DRAWINGS HEAVY VERTICAL
	0x2504: "-",                         // BOX DRAWINGS LIGHT TRIPLE DASH HORIZONTAL
	0x2505: "-",                         // BOX DRAWINGS HEAVY TRIPLE DASH HORIZONTAL
	0x2506: "|",                         // BOX DRAWINGS LIGHT TRIPLE DASH VERTICAL
	0x2507: "|",                         // BOX DRAWINGS HEAVY TRIPLE DASH VERTICAL
	0x2508: "-",                         // BOX DRAWINGS LIGHT QUADRUPLE DASH HORIZONTAL
	0x2509: "-",                         // BOX DRAWINGS HEAVY QUADRUPLE DASH HORIZONTAL
	0x250A: "|",                         // BOX DRAWINGS LIGHT QUADRUPLE DASH VERTICAL
	0x250B: "|",                         // BOX DRAWINGS HEAVY QUADRUPLE DASH VERTICAL
	0x250C: "+",                         // BOX DRAWINGS LIGHT DOWN AND RIGHT
	0x250D: "+",                         // BOX DRAWINGS DOWN LIGHT AND RIGHT HEAVY
	0x250E: "+",                         // BOX DRAWINGS DOWN HEAVY AND RIGHT LIGHT
	0x250F: "+",                         // BOX DRAWINGS HEAVY DOWN AND RIGHT
	0x2510: "+",                         // BOX DRAWINGS LIGHT DOWN AND LEFT
	0x2511: "+",                         // BOX DRAWINGS DOWN LIGHT AND LEFT HEAVY
	0x2512: "+",                         // BOX DRAWINGS DOWN HEAVY AND LEFT LIGHT
	0x2513: "+",                         // BOX DRAWINGS HEAVY DOWN AND LEFT
	0x2514: "+",                         // BOX DRAWINGS LIGHT UP AND RIGHT
	0x2515: "+",                         // BOX DRAWINGS UP LIGHT AND RIGHT HEAVY
	0x2516: "+",                         // BOX DRAWINGS UP HEAVY AND RIGHT LIGHT
	0x2517: "+",                         // BOX DRAWINGS HEAVY UP AND RIGHT
	0x2518: "+",                         // BOX DRAWINGS LIGHT UP AND LEFT
	0x2519: "+",                         // BOX DRAWINGS UP LIGHT AND LEFT HEAVY
	0x251A: "+",                         // BOX DRAWINGS UP HEAVY AND LEFT LIGHT
	0x251B: "+",                         // BOX DRAWINGS HEAVY UP AND LEFT
	0x251C: "+",                         // BOX DRAWINGS LIGHT VERTICAL AND RIGHT
	0x251D: "+",                         // BOX DRAWINGS VERTICAL LIGHT AND RIGHT HEAVY
	0x251E: "+",                         // BOX DRAWINGS UP HEAVY AND RIGHT DOWN LIGHT
	0x251F: "+",                         // BOX DRAWINGS DOWN HEAVY AND RIGHT UP LIGHT
	0x2520: "+",                         // BOX DRAWINGS VERTICAL HEAVY AND RIGHT LIGHT
	0x2521: "+",                         // BOX DRAWINGS DOWN LIGHT AND RIGHT UP HEAVY
	0x2522: "+",                         // BOX DRAWINGS UP LIGHT AND RIGHT DOWN HEAVY
	0x2523: "+",                         // BOX DRAWINGS HEAVY VERTICAL AND RIGHT
	0x2524: "+",                         // BOX DRAWINGS LIGHT VERTICAL AND LEFT
	0x2525: "+",                         // BOX DRAWINGS VERTICAL LIGHT AND LEFT HEAVY
	0x2526: "+",                         // BOX DRAWINGS UP HEAVY AND LEFT DOWN LIGHT
	0x2527: "+",                         // BOX DRAWINGS DOWN HEAVY AND LEFT UP LIGHT
	0x2528: "+",                         // BOX DRAWINGS VERTICAL HEAVY AND LEFT LIGHT
	0x2529: "+",                         // BOX DRAWINGS DOWN LIGHT AND LEFT UP HEAVY
	0x252A: "+",                         // BOX DRAWINGS UP LIGHT AND LEFT DOWN HEAVY
	0x252B: "+",                         // BOX DRAWINGS HEAVY VERTICAL AND LEFT
	0x252C: "+",                         // BOX DRAWINGS LIGHT DOWN AND HORIZONTAL
	0x252D: "+",                         // BOX DRAWINGS LEFT HEAVY AND RIGHT DOWN LIGHT
	0x252E: "+",                         // BOX DRAWINGS RIGHT HEAVY AND LEFT DOWN LIGHT
	0x252F: "+",                         // BOX DRAWINGS DOWN LIGHT AND HORIZONTAL HEAVY
	0x2530: "+",                         // BOX DRAWINGS DOWN HEAVY AND HORIZONTAL LIGHT
	0x2531: "+",                         // BOX DRAWINGS RIGHT LIGHT AND LEFT DOWN HEAVY
	0x2532: "+",                         // BOX DRAWINGS LEFT LIGHT AND RIGHT DOWN HEAVY
	0x2533: "+",                         // BOX DRAWINGS HEAVY DOWN AND HORIZONTAL
	0x2534: "+",                         // BOX DRAWINGS LIGHT UP AND HORIZONTAL
	0x2535: "+",                         // BOX DRAWINGS LEFT HEAVY AND RIGHT UP LIGHT
	0x2536: "+",                         // BOX DRAWINGS RIGHT HEAVY AND LEFT UP LIGHT
	0x2537: "+",                         // BOX DRAWINGS UP LIGHT AND HORIZONTAL HEAVY
	0x2538: "+",                         // BOX DRAWINGS UP HEAVY AND HORIZONTAL LIGHT
	0x2539: "+",                         // BOX DRAWINGS RIGHT LIGHT AND LEFT UP HEAVY
	0x253A: "+",                         // BOX DRAWINGS LEFT LIGHT AND RIGHT UP HEAVY
	0x253B: "+",                         // BOX DRAWINGS HEAVY UP AND HORIZONTAL
	0x253C: "+",                         // BOX DRAWINGS LIGHT VERTICAL AND HORIZONTAL
	0x253D: "+",                         // BOX DRAWINGS LEFT HEAVY AND RIGHT VERTICAL LIGHT
	0x253E: "+",                         // BOX DRAWINGS RIGHT HEAVY AND LEFT VERTICAL LIGHT
	0x253F: "+",                         // BOX DRAWINGS VERTICAL LIGHT AND HORIZONTAL HEAVY
	0x2540: "+",                         // BOX DRAWINGS UP HEAVY AND DOWN HORIZONTAL LIGHT
	0x2541: "+",                         // BOX DRAWINGS DOWN HEAVY AND UP HORIZONTAL LIGHT
	0x2542: "+",                         // BOX DRAWINGS VERTICAL HEAVY AND HORIZONTAL LIGHT
	0x2543: "+",                         // BOX DRAWINGS LEFT UP HEAVY AND RIGHT DOWN LIGHT
	0x2544: "+",                         // BOX DRAWINGS RIGHT UP HEAVY AND LEFT DOWN LIGHT
	0x2545: "+",                         // BOX DRAWINGS LEFT DOWN HEAVY AND RIGHT UP LIGHT
	0x2546: "+",                         // BOX DRAWINGS RIGHT DOWN HEAVY AND LEFT UP LIGHT
	0x2547: "+",                         // BOX DRAWINGS DOWN LIGHT AND UP HORIZONTAL HEAVY
	0x2548: "+",                         // BOX DRAWINGS UP LIGHT AND DOWN HORIZONTAL HEAVY
	0x2549: "+",                         // BOX DRAWINGS RIGHT LIGHT AND LEFT VERTICAL HEAVY
	0x254A: "+",                         // BOX DRAWINGS LEFT LIGHT AND RIGHT VERTICAL HEAVY
	0x254B: "+",                         // BOX DRAWINGS HEAVY VERTICAL AND HORIZONTAL
	0x254C: "-",                         // BOX DRAWINGS LIGHT DOUBLE DASH HORIZONTAL
	0x254D: "-",                         // BOX DRAWINGS HEAVY DOUBLE DASH HORIZONTAL
	0x254E: "|",                         // BOX DRAWINGS LIGHT DOUBLE DASH VERTICAL
	0x254F: "|",                         // BOX DRAWINGS HEAVY DOUBLE DASH VERTICAL
	0x2550: "-",                         // BOX DRAWINGS DOUBLE HORIZONTAL
	0x2551: "|",                         // BOX DRAWINGS DOUBLE VERTICAL
	0x2552: "+",                         // BOX DRAWINGS DOWN SINGLE AND RIGHT DOUBLE
	0x2553: "+",                         // BOX DRAWINGS DOWN DOUBLE AND RIGHT SINGLE
	0x2554: "+",                         // BOX DRAWINGS DOUBLE DOWN AND RIGHT
	0x2555: "+",                         // BOX DRAWINGS DOWN SINGLE AND LEFT DOUBLE
	0x2556: "+",                         // BOX DRAWINGS DOWN DOUBLE AND LEFT SINGLE
	0x2557: "+",                         // BOX DRAWINGS DOUBLE DOWN AND LEFT
	0x2558: "+",                         // BOX DRAWINGS UP SINGLE AND RIGHT DOUBLE
	0x2559: "+",                         // BOX DRAWINGS UP DOUBLE AND RIGHT SINGLE
	0x255A: "+",                         // BOX DRAWINGS DOUBLE UP AND RIGHT
	0x255B: "+",                         // BOX DRAWINGS UP SINGLE AND LEFT DOUBLE
	0x255C: "+",                         // BOX DRAWINGS UP DOUBLE AND LEFT SINGLE
	0x255D: "+",                         // BOX DRAWINGS DOUBLE UP AND LEFT
	0x255E: "+",                         // BOX DRAWINGS VERTICAL SINGLE AND RIGHT DOUBLE
	0x255F: "+",                         // BOX DRAWINGS VERTICAL DOUBLE AND RIGHT SINGLE
	0x2560: "+",                         // BOX DRAWINGS DOUBLE VERTICAL AND RIGHT
	0x2561: "+",                         // BOX DRAWINGS VERTICAL SINGLE AND LEFT DOUBLE
	0x2562: "+",                         // BOX DRAWINGS VERTICAL DOUBLE AND LEFT SINGLE
	0x2563: "+",                         // BOX DRAWINGS DOUBLE VERTICAL AND LEFT
	0x2564: "+",                         // BOX DRAWINGS DOWN SINGLE AND HORIZONTAL DOUBLE
	0x2565: "+",                         // BOX DRAWINGS DOWN DOUBLE AND HORIZONTAL SINGLE
	0x2566: "+",                         // BOX DRAWINGS DOUBLE DOWN AND HORIZONTAL
	0x2567: "+",                         // BOX DRAWINGS UP SINGLE AND HORIZONTAL DOUBLE
	0x2568: "+",                         // BOX DRAWINGS UP DOUBLE AND HORIZONTAL SINGLE
	0x2569: "+",                         // BOX DRAWINGS DOUBLE UP AND HORIZONTAL
	0x256A: "+",                         // BOX DRAWINGS VERTICAL SINGLE AND HORIZONTAL DOUBLE
	0x256B: "+",                         // BOX DRAWINGS VERTICAL DOUBLE AND HORIZONTAL SINGLE
	0x256C: "+",                         // BOX DRAWINGS DOUBLE VERTICAL AND HORIZONTAL
	0x256D: "+",                         // BOX DRAWINGS LIGHT ARC DOWN AND RIGHT
	0x256E: "+",                         // BOX DRAWINGS LIGHT ARC DOWN AND LEFT
	0x256F: "+",                         // BOX DRAWINGS LIGHT ARC UP AND LEFT
	0x2570: "+",                         // BOX DRAWINGS LIGHT ARC UP AND RIGHT
	0x2571: "/",                         // BOX DRAWINGS LIGHT DIAGONAL UPPER RIGHT TO LOWER LEFT
	0x2572: "\\",                        // BOX DRAWINGS LIGHT DIAGONAL UPPER LEFT TO LOWER RIGHT
	0x2573: "X",                         // BOX DRAWINGS LIGHT DIAGONAL CROSS
	0x2574: "-",                         // BOX DRAWINGS LIGHT LEFT
	0x2575: "|",                         // BOX DRAWINGS LIGHT UP
	0x2576: "-",                         // BOX DRAWINGS LIGHT RIGHT
	0x2577: "|",                         // BOX DRAWINGS LIGHT DOWN
	0x2578: "-",                         // BOX DRAWINGS HEAVY LEFT
	0x2579: "|",                         // BOX DRAWINGS HEAVY UP
	0x257A: "-",                         // BOX DRAWINGS HEAVY RIGHT
	0x257B: "|",                         // BOX DRAWINGS HEAVY DOWN
	0x257C: "-",                         // BOX DRAWINGS LIGHT LEFT AND HEAVY RIGHT
	0x257D: "|",                         // BOX DRAWINGS LIGHT UP AND HEAVY DOWN
	0x257E: "-",                         // BOX DRAWINGS HEAVY LEFT AND LIGHT RIGHT
	0x257F: "|",                         // BOX DRAWINGS HEAVY UP AND LIGHT DOWN
	0x2580: "#",                         // UPPER HALF BLOCK
	0x2581: "#",                         // LOWER ONE EIGHTH BLOCK
	0x2582: "#",                         // LOWER ONE QUARTER BLOCK
	0x2583: "#",                         // LOWER THREE EIGHTHS BLOCK
	0x2584: "#",                         // LOWER HALF BLOCK
	0x2585: "#",                         // LOWER FIVE EIGHTHS BLOCK
	0x2586: "#",                         // LOWER THREE QUARTERS BLOCK
	0x2587: "#",                         // LOWER SEVEN EIGHTHS BLOCK
	0x2588: "#",                         // FULL BLOCK
	0x2589: "#",                         // LEFT SEVEN EIGHTHS BLOCK
	0x258A: "#",                         // LEFT THREE QUARTERS BLOCK
	0x258B: "#",                         // LEFT FIVE EIGHTHS BLOCK
	0x258C: "#",                         // LEFT HALF BLOCK
	0x258D: "#",                         // LEFT THREE EIGHTHS BLOCK
	0x258E: "#",                         // LEFT ONE QUARTER BLOCK
	0x258F: "#",                         // LEFT ONE EIGHTH BLOCK
	0x2590: "#",                         // RIGHT HALF BLOCK
	0x2591: "#",                         // LIGHT SHADE
	0x2592: "#",                         // MEDIUM SHADE
	0x2593: "#",                         // DARK SHADE
	0x2594: "-",                         // UPPER ONE EIGHTH BLOCK
	0x2595: "|",                         // RIGHT ONE EIGHTH BLOCK
	0x25A0: "#",                         // BLACK SQUARE
	0x25A1: "#",                         // WHITE SQUARE
	0x25A2: "#",                         // WHITE SQUARE WITH ROUNDED CORNERS
	0x25A3: "#",                         // WHITE SQUARE CONTAINING BLACK SMALL SQUARE
	0x25A4: "#",                         // SQUARE WITH HORIZONTAL FILL
	0x25A5: "#",                         // SQUARE WITH VERTICAL FILL
	0x25A6: "#",                         // SQUARE WITH ORTHOGONAL CROSSHATCH FILL
	0x25A7: "#",                         // SQUARE WITH UPPER LEFT TO LOWER RIGHT FILL
	0x25A8: "#",                         // SQUARE WITH UPPER RIGHT TO LOWER LEFT FILL
	0x25A9: "#",                         // SQUARE WITH DIAGONAL CROSSHATCH FILL
	0x25AA: "#",                         // BLACK SMALL SQUARE
	0x25AB: "#",                         // WHITE SMALL SQUARE
	0x25AC: "#",                         // BLACK RECTANGLE
	0x25AD: "#",                         // WHITE RECTANGLE
	0x25AE: "#",                         // BLACK VERTICAL RECTANGLE
	0x25AF: "#",                         // WHITE VERTICAL RECTANGLE
	0x25B0: "#",                         // BLACK PARALLELOGRAM
	0x25B1: "#",                         // WHITE PARALLELOGRAM
	0x25B2: "^",                         // BLACK UP-POINTING TRIANGLE
	0x25B3: "^",                         // WHITE UP-POINTING TRIANGLE
	0x25B4: "^",                         // BLACK UP-POINTING SMALL TRIANGLE
	0x25B5: "^",                         // WHITE UP-POINTING SMALL TRIANGLE
	0x25B6: ">",                         // BLACK RIGHT-POINTING TRIANGLE
	0x25B7: ">",                         // WHITE RIGHT-POINTING TRIANGLE
	0x25B8: ">",                         // BLACK RIGHT-POINTING SMALL TRIANGLE
	0x25B9: ">",                         // WHITE RIGHT-POINTING SMALL TRIANGLE
	0x25BA: ">",                         // BLACK RIGHT-POINTING POINTER
	0x25BB: ">",                         // WHITE RIGHT-POINTING POINTER
	0x25BC: "V",                         // BLACK DOWN-POINTING TRIANGLE
	0x25BD: "V",                         // WHITE DOWN-POINTING TRIANGLE
	0x25BE: "V",                         // BLACK DOWN-POINTING SMALL TRIANGLE
	0x25BF: "V",                         // WHITE DOWN-POINTING SMALL TRIANGLE
	0x25C0: "<",                         // BLACK LEFT-POINTING TRIANGLE
	0x25C1: "<",                         // WHITE LEFT-POINTING TRIANGLE
	0x25C2: "<",                         // BLACK LEFT-POINTING SMALL TRIANGLE
	0x25C3: "<",                         // WHITE LEFT-POINTING SMALL TRIANGLE
	0x25C4: "<",                         // BLACK LEFT-POINTING POINTER
	0x25C5: "<",                         // WHITE LEFT-POINTING POINTER
	0x25C6: "*",                         // BLACK DIAMOND
	0x25C7: "*",                         // WHITE DIAMOND
	0x25C8: "*",                         // WHITE DIAMOND CONTAINING BLACK SMALL DIAMOND
	0x25C9: "*",                         // FISHEYE
	0x25CA: "*",                         // LOZENGE
	0x25CB: "*",                         // WHITE CIRCLE
	0x25CC: "*",                         // DOTTED CIRCLE
	0x25CD: "*",                         // CIRCLE WITH VERTICAL FILL
	0x25CE: "*",                         // BULLSEYE
	0x25CF: "*",                         // BLACK CIRCLE
	0x25D0: "*",                         // CIRCLE WITH LEFT HALF BLACK
	0x25D1: "*",                         // CIRCLE WITH RIGHT HALF BLACK
	0x25D2: "*",                         // CIRCLE WITH LOWER HALF BLACK
	0x25D3: "*",                         // CIRCLE WITH UPPER HALF BLACK
	0x25D4: "*",                         // CIRCLE WITH UPPER RIGHT QUADRANT BLACK
	0x25D5: "*",                         // CIRCLE WITH ALL BUT UPPER LEFT QUADRANT BLACK
	0x25D6: "*",                         // LEFT HALF BLACK CIRCLE
	0x25D7: "*",                         // RIGHT HALF BLACK CIRCLE
	0x25D8: "*",                         // INVERSE BULLET
	0x25D9: "*",                         // INVERSE WHITE CIRCLE
	0x25DA: "*",                         // UPPER HALF INVERSE WHITE CIRCLE
	0x25DB: "*",                         // LOWER HALF INVERSE WHITE CIRCLE
	0x25DC: "*",                         // UPPER LEFT QUADRANT CIRCULAR ARC
	0x25DD: "*",                         // UPPER RIGHT QUADRANT CIRCULAR ARC
	0x25DE: "*",                         // LOWER RIGHT QUADRANT CIRCULAR ARC
	0x25DF: "*",                         // LOWER LEFT QUADRANT CIRCULAR ARC
	0x25E0: "*",                         // UPPER HALF CIRCLE
	0x25E1: "*",                         // LOWER HALF CIRCLE
	0x25E2: "*",                         // BLACK LOWER RIGHT TRIANGLE
	0x25E3: "*",                         // BLACK LOWER LEFT TRIANGLE
	0x25E4: "*",                         // BLACK UPPER LEFT TRIANGLE
	0x25E5: "*",                         // BLACK UPPER RIGHT TRIANGLE
	0x25E6: "*",                         // WHITE BULLET
	0x25E7: "#",                         // SQUARE WITH LEFT HALF BLACK
	0x25E8: "#",                         // SQUARE WITH RIGHT HALF BLACK
	0x25E9: "#",                         // SQUARE WITH UPPER LEFT DIAGONAL HALF BLACK
	0x25EA: "#",                         // SQUARE WITH LOWER RIGHT DIAGONAL HALF BLACK
	0x25EB: "#",                         // WHITE SQUARE WITH VERTICAL BISECTING LINE
	0x25EC: "^",                         // WHITE UP-POINTING TRIANGLE WITH DOT
	0x25ED: "^",                         // UP-POINTING TRIANGLE WITH LEFT HALF BLACK
	0x25EE: "^",                         // UP-POINTING TRIANGLE WITH RIGHT HALF BLACK
	0x25EF: "O",                         // LARGE CIRCLE
	0x25F0: "#",                         // WHITE SQUARE WITH UPPER LEFT QUADRANT
	0x25F1: "#",                         // WHITE SQUARE WITH LOWER LEFT QUADRANT
	0x25F2: "#",                         // WHITE SQUARE WITH LOWER RIGHT QUADRANT
	0x25F3: "#",                         // WHITE SQUARE WITH UPPER RIGHT QUADRANT
	0x25F4: "#",                         // WHITE CIRCLE WITH UPPER LEFT QUADRANT
	0x25F5: "#",                         // WHITE CIRCLE WITH LOWER LEFT QUADRANT
	0x25F6: "#",                         // WHITE CIRCLE WITH LOWER RIGHT QUADRANT
	0x25F7: "#",                         // WHITE CIRCLE WITH UPPER RIGHT QUADRANT
	0x2605: " (starf) )",                // BLACK STAR
	0x2606: " (star) ",                  // WHITE STAR
	0x260E: " (phone) ",                 // BLACK TELEPHONE
	0x2640: " (female) ",                // FEMALE SIGN
	0x2642: " (male) ",                  // MALE SIGN
	0x2660: " (spades) ",                // BLACK SPADE SUIT
	0x2661: " (hearts) ",                // WHITE HEART SUIT
	0x2662: " (diamonds) ",              // WHITE DIAMOND SUIT
	0x2663: " (clubs) ",                 // BLACK CLUB SUIT
	0x2664: " (spades) ",                // WHITE SPADE SUIT
	0x2665: " (hearts) ",                // BLACK HEART SUIT
	0x2666: " (diamonds) ",              // BLACK DIAMOND SUIT
	0x2667: " (clubs) ",                 // WHITE CLUB SUIT
	0x2669: " (music note) ",            // QUARTER NOTE
	0x266A: " (sung) ",                  // EIGHTH NOTE
	0x266B: " (music note) ",            // BEAMED EIGHTH NOTES
	0x266C: " (music note) ",            // BEAMED SIXTEENTH NOTES
	0x266D: " (flat) ",                  // MUSIC FLAT SIGN
	0x266E: " (natural) ",               // MUSIC NATURAL SIGN
	0x266F: " (sharp) ",                 // MUSIC SHARP SIGN
	0x2713: " (check) ",                 // CHECK MARK
	0x2714: " (check) ",                 // HEAVY CHECK MARK
	0x2717: " (cross) ",                 // BALLOT X
	0x2720: " (malt) ",                  // MALTESE CROSS
	0x2731: "*",                         // HEAVY ASTERISK
	0x2736: " (sext) ",                  // SIX POINTED BLACK STAR
	0x2758: "|",                         // LIGHT VERTICAL BAR
	0x2762: "!",                         // HEAVY EXCLAMATION MARK ORNAMENT
	0x27E6: "[",                         //
	0x27E7: "]",                         //
	0x27E8: "<",                         //
	0x27E9: "> ",                        //
	0x2801: "a",                         // BRAILLE PATTERN DOTS-1
	0x2802: "1",                         // BRAILLE PATTERN DOTS-2
	0x2803: "b",                         // BRAILLE PATTERN DOTS-12
	0x2804: "'",                         // BRAILLE PATTERN DOTS-3
	0x2805: "k",                         // BRAILLE PATTERN DOTS-13
	0x2806: "2",                         // BRAILLE PATTERN DOTS-23
	0x2807: "l",                         // BRAILLE PATTERN DOTS-123
	0x2808: "@",                         // BRAILLE PATTERN DOTS-4
	0x2809: "c",                         // BRAILLE PATTERN DOTS-14
	0x280A: "i",                         // BRAILLE PATTERN DOTS-24
	0x280B: "f",                         // BRAILLE PATTERN DOTS-124
	0x280C: "/",                         // BRAILLE PATTERN DOTS-34
	0x280D: "m",                         // BRAILLE PATTERN DOTS-134
	0x280E: "s",                         // BRAILLE PATTERN DOTS-234
	0x280F: "p",                         // BRAILLE PATTERN DOTS-1234
	0x2810: "",                          // BRAILLE PATTERN DOTS-5
	0x2811: "e",                         // BRAILLE PATTERN DOTS-15
	0x2812: "3",                         // BRAILLE PATTERN DOTS-25
	0x2813: "h",                         // BRAILLE PATTERN DOTS-125
	0x2814: "9",                         // BRAILLE PATTERN DOTS-35
	0x2815: "o",                         // BRAILLE PATTERN DOTS-135
	0x2816: "6",                         // BRAILLE PATTERN DOTS-235
	0x2817: "r",                         // BRAILLE PATTERN DOTS-1235
	0x2818: "^",                         // BRAILLE PATTERN DOTS-45
	0x2819: "d",                         // BRAILLE PATTERN DOTS-145
	0x281A: "j",                         // BRAILLE PATTERN DOTS-245
	0x281B: "g",                         // BRAILLE PATTERN DOTS-1245
	0x281C: ">",                         // BRAILLE PATTERN DOTS-345
	0x281D: "n",                         // BRAILLE PATTERN DOTS-1345
	0x281E: "t",                         // BRAILLE PATTERN DOTS-2345
	0x281F: "q",                         // BRAILLE PATTERN DOTS-12345
	0x2820: ",",                         // BRAILLE PATTERN DOTS-6
	0x2821: "*",                         // BRAILLE PATTERN DOTS-16
	0x2822: "5",                         // BRAILLE PATTERN DOTS-26
	0x2823: "<",                         // BRAILLE PATTERN DOTS-126
	0x2824: "-",                         // BRAILLE PATTERN DOTS-36
	0x2825: "u",                         // BRAILLE PATTERN DOTS-136
	0x2826: "8",                         // BRAILLE PATTERN DOTS-236
	0x2827: "v",                         // BRAILLE PATTERN DOTS-1236
	0x2828: ".",                         // BRAILLE PATTERN DOTS-46
	0x2829: "%",                         // BRAILLE PATTERN DOTS-146
	0x282A: "[",                         // BRAILLE PATTERN DOTS-246
	0x282B: "$",                         // BRAILLE PATTERN DOTS-1246
	0x282C: "+",                         // BRAILLE PATTERN DOTS-346
	0x282D: "x",                         // BRAILLE PATTERN DOTS-1346
	0x282E: "!",                         // BRAILLE PATTERN DOTS-2346
	0x282F: "&",                         // BRAILLE PATTERN DOTS-12346
	0x2830: ";",                         // BRAILLE PATTERN DOTS-56
	0x2831: ":",                         // BRAILLE PATTERN DOTS-156
	0x2832: "4",                         // BRAILLE PATTERN DOTS-256
	0x2833: "\\",                        // BRAILLE PATTERN DOTS-1256
	0x2834: "0",                         // BRAILLE PATTERN DOTS-356
	0x2835: "z",                         // BRAILLE PATTERN DOTS-1356
	0x2836: "7",                         // BRAILLE PATTERN DOTS-2356
	0x2837: "(",                         // BRAILLE PATTERN DOTS-12356
	0x2838: "_",                         // BRAILLE PATTERN DOTS-456
	0x2839: "?",                         // BRAILLE PATTERN DOTS-1456
	0x283A: "w",                         // BRAILLE PATTERN DOTS-2456
	0x283B: "]",                         // BRAILLE PATTERN DOTS-12456
	0x283C: "#",                         // BRAILLE PATTERN DOTS-3456
	0x283D: "y",                         // BRAILLE PATTERN DOTS-13456
	0x283E: ")",                         // BRAILLE PATTERN DOTS-23456
	0x283F: "=",                         // BRAILLE PATTERN DOTS-123456
	0x2840: "[d7]",                      // BRAILLE PATTERN DOTS-7
	0x2841: "[d17]",                     // BRAILLE PATTERN DOTS-17
	0x2842: "[d27]",                     // BRAILLE PATTERN DOTS-27
	0x2843: "[d127]",                    // BRAILLE PATTERN DOTS-127
	0x2844: "[d37]",                     // BRAILLE PATTERN DOTS-37
	0x2845: "[d137]",                    // BRAILLE PATTERN DOTS-137
	0x2846: "[d237]",                    // BRAILLE PATTERN DOTS-237
	0x2847: "[d1237]",                   // BRAILLE PATTERN DOTS-1237
	0x2848: "[d47]",                     // BRAILLE PATTERN DOTS-47
	0x2849: "[d147]",                    // BRAILLE PATTERN DOTS-147
	0x284A: "[d247]",                    // BRAILLE PATTERN DOTS-247
	0x284B: "[d1247]",                   // BRAILLE PATTERN DOTS-1247
	0x284C: "[d347]",                    // BRAILLE PATTERN DOTS-347
	0x284D: "[d1347]",                   // BRAILLE PATTERN DOTS-1347
	0x284E: "[d2347]",                   // BRAILLE PATTERN DOTS-2347
	0x284F: "[d12347]",                  // BRAILLE PATTERN DOTS-12347
	0x2850: "[d57]",                     // BRAILLE PATTERN DOTS-57
	0x2851: "[d157]",                    // BRAILLE PATTERN DOTS-157
	0x2852: "[d257]",                    // BRAILLE PATTERN DOTS-257
	0x2853: "[d1257]",                   // BRAILLE PATTERN DOTS-1257
	0x2854: "[d357]",                    // BRAILLE PATTERN DOTS-357
	0x2855: "[d1357]",                   // BRAILLE PATTERN DOTS-1357
	0x2856: "[d2357]",                   // BRAILLE PATTERN DOTS-2357
	0x2857: "[d12357]",                  // BRAILLE PATTERN DOTS-12357
	0x2858: "[d457]",                    // BRAILLE PATTERN DOTS-457
	0x2859: "[d1457]",                   // BRAILLE PATTERN DOTS-1457
	0x285A: "[d2457]",                   // BRAILLE PATTERN DOTS-2457
	0x285B: "[d12457]",                  // BRAILLE PATTERN DOTS-12457
	0x285C: "[d3457]",                   // BRAILLE PATTERN DOTS-3457
	0x285D: "[d13457]",                  // BRAILLE PATTERN DOTS-13457
	0x285E: "[d23457]",                  // BRAILLE PATTERN DOTS-23457
	0x285F: "[d123457]",                 // BRAILLE PATTERN DOTS-123457
	0x2860: "[d67]",                     // BRAILLE PATTERN DOTS-67
	0x2861: "[d167]",                    // BRAILLE PATTERN DOTS-167
	0x2862: "[d267]",                    // BRAILLE PATTERN DOTS-267
	0x2863: "[d1267]",                   // BRAILLE PATTERN DOTS-1267
	0x2864: "[d367]",                    // BRAILLE PATTERN DOTS-367
	0x2865: "[d1367]",                   // BRAILLE PATTERN DOTS-1367
	0x2866: "[d2367]",                   // BRAILLE PATTERN DOTS-2367
	0x2867: "[d12367]",                  // BRAILLE PATTERN DOTS-12367
	0x2868: "[d467]",                    // BRAILLE PATTERN DOTS-467
	0x2869: "[d1467]",                   // BRAILLE PATTERN DOTS-1467
	0x286A: "[d2467]",                   // BRAILLE PATTERN DOTS-2467
	0x286B: "[d12467]",                  // BRAILLE PATTERN DOTS-12467
	0x286C: "[d3467]",                   // BRAILLE PATTERN DOTS-3467
	0x286D: "[d13467]",                  // BRAILLE PATTERN DOTS-13467
	0x286E: "[d23467]",                  // BRAILLE PATTERN DOTS-23467
	0x286F: "[d123467]",                 // BRAILLE PATTERN DOTS-123467
	0x2870: "[d567]",                    // BRAILLE PATTERN DOTS-567
	0x2871: "[d1567]",                   // BRAILLE PATTERN DOTS-1567
	0x2872: "[d2567]",                   // BRAILLE PATTERN DOTS-2567
	0x2873: "[d12567]",                  // BRAILLE PATTERN DOTS-12567
	0x2874: "[d3567]",                   // BRAILLE PATTERN DOTS-3567
	0x2875: "[d13567]",                  // BRAILLE PATTERN DOTS-13567
	0x2876: "[d23567]",                  // BRAILLE PATTERN DOTS-23567
	0x2877: "[d123567]",                 // BRAILLE PATTERN DOTS-123567
	0x2878: "[d4567]",                   // BRAILLE PATTERN DOTS-4567
	0x2879: "[d14567]",                  // BRAILLE PATTERN DOTS-14567
	0x287A: "[d24567]",                  // BRAILLE PATTERN DOTS-24567
	0x287B: "[d124567]",                 // BRAILLE PATTERN DOTS-124567
	0x287C: "[d34567]",                  // BRAILLE PATTERN DOTS-34567
	0x287D: "[d134567]",                 // BRAILLE PATTERN DOTS-134567
	0x287E: "[d234567]",                 // BRAILLE PATTERN DOTS-234567
	0x287F: "[d1234567]",                // BRAILLE PATTERN DOTS-1234567
	0x2880: "[d8]",                      // BRAILLE PATTERN DOTS-8
	0x2881: "[d18]",                     // BRAILLE PATTERN DOTS-18
	0x2882: "[d28]",                     // BRAILLE PATTERN DOTS-28
	0x2883: "[d128]",                    // BRAILLE PATTERN DOTS-128
	0x2884: "[d38]",                     // BRAILLE PATTERN DOTS-38
	0x2885: "[d138]",                    // BRAILLE PATTERN DOTS-138
	0x2886: "[d238]",                    // BRAILLE PATTERN DOTS-238
	0x2887: "[d1238]",                   // BRAILLE PATTERN DOTS-1238
	0x2888: "[d48]",                     // BRAILLE PATTERN DOTS-48
	0x2889: "[d148]",                    // BRAILLE PATTERN DOTS-148
	0x288A: "[d248]",                    // BRAILLE PATTERN DOTS-248
	0x288B: "[d1248]",                   // BRAILLE PATTERN DOTS-1248
	0x288C: "[d348]",                    // BRAILLE PATTERN DOTS-348
	0x288D: "[d1348]",                   // BRAILLE PATTERN DOTS-1348
	0x288E: "[d2348]",                   // BRAILLE PATTERN DOTS-2348
	0x288F: "[d12348]",                  // BRAILLE PATTERN DOTS-12348
	0x2890: "[d58]",                     // BRAILLE PATTERN DOTS-58
	0x2891: "[d158]",                    // BRAILLE PATTERN DOTS-158
	0x2892: "[d258]",                    // BRAILLE PATTERN DOTS-258
	0x2893: "[d1258]",                   // BRAILLE PATTERN DOTS-1258
	0x2894: "[d358]",                    // BRAILLE PATTERN DOTS-358
	0x2895: "[d1358]",                   // BRAILLE PATTERN DOTS-1358
	0x2896: "[d2358]",                   // BRAILLE PATTERN DOTS-2358
	0x2897: "[d12358]",                  // BRAILLE PATTERN DOTS-12358
	0x2898: "[d458]",                    // BRAILLE PATTERN DOTS-458
	0x2899: "[d1458]",                   // BRAILLE PATTERN DOTS-1458
	0x289A: "[d2458]",                   // BRAILLE PATTERN DOTS-2458
	0x289B: "[d12458]",                  // BRAILLE PATTERN DOTS-12458
	0x289C: "[d3458]",                   // BRAILLE PATTERN DOTS-3458
	0x289D: "[d13458]",                  // BRAILLE PATTERN DOTS-13458
	0x289E: "[d23458]",                  // BRAILLE PATTERN DOTS-23458
	0x289F: "[d123458]",                 // BRAILLE PATTERN DOTS-123458
	0x28A0: "[d68]",                     // BRAILLE PATTERN DOTS-68
	0x28A1: "[d168]",                    // BRAILLE PATTERN DOTS-168
	0x28A2: "[d268]",                    // BRAILLE PATTERN DOTS-268
	0x28A3: "[d1268]",                   // BRAILLE PATTERN DOTS-1268
	0x28A4: "[d368]",                    // BRAILLE PATTERN DOTS-368
	0x28A5: "[d1368]",                   // BRAILLE PATTERN DOTS-1368
	0x28A6: "[d2368]",                   // BRAILLE PATTERN DOTS-2368
	0x28A7: "[d12368]",                  // BRAILLE PATTERN DOTS-12368
	0x28A8: "[d468]",                    // BRAILLE PATTERN DOTS-468
	0x28A9: "[d1468]",                   // BRAILLE PATTERN DOTS-1468
	0x28AA: "[d2468]",                   // BRAILLE PATTERN DOTS-2468
	0x28AB: "[d12468]",                  // BRAILLE PATTERN DOTS-12468
	0x28AC: "[d3468]",                   // BRAILLE PATTERN DOTS-3468
	0x28AD: "[d13468]",                  // BRAILLE PATTERN DOTS-13468
	0x28AE: "[d23468]",                  // BRAILLE PATTERN DOTS-23468
	0x28AF: "[d123468]",                 // BRAILLE PATTERN DOTS-123468
	0x28B0: "[d568]",                    // BRAILLE PATTERN DOTS-568
	0x28B1: "[d1568]",                   // BRAILLE PATTERN DOTS-1568
	0x28B2: "[d2568]",                   // BRAILLE PATTERN DOTS-2568
	0x28B3: "[d12568]",                  // BRAILLE PATTERN DOTS-12568
	0x28B4: "[d3568]",                   // BRAILLE PATTERN DOTS-3568
	0x28B5: "[d13568]",                  // BRAILLE PATTERN DOTS-13568
	0x28B6: "[d23568]",                  // BRAILLE PATTERN DOTS-23568
	0x28B7: "[d123568]",                 // BRAILLE PATTERN DOTS-123568
	0x28B8: "[d4568]",                   // BRAILLE PATTERN DOTS-4568
	0x28B9: "[d14568]",                  // BRAILLE PATTERN DOTS-14568
	0x28BA: "[d24568]",                  // BRAILLE PATTERN DOTS-24568
	0x28BB: "[d124568]",                 // BRAILLE PATTERN DOTS-124568
	0x28BC: "[d34568]",                  // BRAILLE PATTERN DOTS-34568
	0x28BD: "[d134568]",                 // BRAILLE PATTERN DOTS-134568
	0x28BE: "[d234568]",                 // BRAILLE PATTERN DOTS-234568
	0x28BF: "[d1234568]",                // BRAILLE PATTERN DOTS-1234568
	0x28C0: "[d78]",                     // BRAILLE PATTERN DOTS-78
	0x28C1: "[d178]",                    // BRAILLE PATTERN DOTS-178
	0x28C2: "[d278]",                    // BRAILLE PATTERN DOTS-278
	0x28C3: "[d1278]",                   // BRAILLE PATTERN DOTS-1278
	0x28C4: "[d378]",                    // BRAILLE PATTERN DOTS-378
	0x28C5: "[d1378]",                   // BRAILLE PATTERN DOTS-1378
	0x28C6: "[d2378]",                   // BRAILLE PATTERN DOTS-2378
	0x28C7: "[d12378]",                  // BRAILLE PATTERN DOTS-12378
	0x28C8: "[d478]",                    // BRAILLE PATTERN DOTS-478
	0x28C9: "[d1478]",                   // BRAILLE PATTERN DOTS-1478
	0x28CA: "[d2478]",                   // BRAILLE PATTERN DOTS-2478
	0x28CB: "[d12478]",                  // BRAILLE PATTERN DOTS-12478
	0x28CC: "[d3478]",                   // BRAILLE PATTERN DOTS-3478
	0x28CD: "[d13478]",                  // BRAILLE PATTERN DOTS-13478
	0x28CE: "[d23478]",                  // BRAILLE PATTERN DOTS-23478
	0x28CF: "[d123478]",                 // BRAILLE PATTERN DOTS-123478
	0x28D0: "[d578]",                    // BRAILLE PATTERN DOTS-578
	0x28D1: "[d1578]",                   // BRAILLE PATTERN DOTS-1578
	0x28D2: "[d2578]",                   // BRAILLE PATTERN DOTS-2578
	0x28D3: "[d12578]",                  // BRAILLE PATTERN DOTS-12578
	0x28D4: "[d3578]",                   // BRAILLE PATTERN DOTS-3578
	0x28D5: "[d13578]",                  // BRAILLE PATTERN DOTS-13578
	0x28D6: "[d23578]",                  // BRAILLE PATTERN DOTS-23578
	0x28D7: "[d123578]",                 // BRAILLE PATTERN DOTS-123578
	0x28D8: "[d4578]",                   // BRAILLE PATTERN DOTS-4578
	0x28D9: "[d14578]",                  // BRAILLE PATTERN DOTS-14578
	0x28DA: "[d24578]",                  // BRAILLE PATTERN DOTS-24578
	0x28DB: "[d124578]",                 // BRAILLE PATTERN DOTS-124578
	0x28DC: "[d34578]",                  // BRAILLE PATTERN DOTS-34578
	0x28DD: "[d134578]",                 // BRAILLE PATTERN DOTS-134578
	0x28DE: "[d234578]",                 // BRAILLE PATTERN DOTS-234578
	0x28DF: "[d1234578]",                // BRAILLE PATTERN DOTS-1234578
	0x28E0: "[d678]",                    // BRAILLE PATTERN DOTS-678
	0x28E1: "[d1678]",                   // BRAILLE PATTERN DOTS-1678
	0x28E2: "[d2678]",                   // BRAILLE PATTERN DOTS-2678
	0x28E3: "[d12678]",                  // BRAILLE PATTERN DOTS-12678
	0x28E4: "[d3678]",                   // BRAILLE PATTERN DOTS-3678
	0x28E5: "[d13678]",                  // BRAILLE PATTERN DOTS-13678
	0x28E6: "[d23678]",                  // BRAILLE PATTERN DOTS-23678
	0x28E7: "[d123678]",                 // BRAILLE PATTERN DOTS-123678
	0x28E8: "[d4678]",                   // BRAILLE PATTERN DOTS-4678
	0x28E9: "[d14678]",                  // BRAILLE PATTERN DOTS-14678
	0x28EA: "[d24678]",                  // BRAILLE PATTERN DOTS-24678
	0x28EB: "[d124678]",                 // BRAILLE PATTERN DOTS-124678
	0x28EC: "[d34678]",                  // BRAILLE PATTERN DOTS-34678
	0x28ED: "[d134678]",                 // BRAILLE PATTERN DOTS-134678
	0x28EE: "[d234678]",                 // BRAILLE PATTERN DOTS-234678
	0x28EF: "[d1234678]",                // BRAILLE PATTERN DOTS-1234678
	0x28F0: "[d5678]",                   // BRAILLE PATTERN DOTS-5678
	0x28F1: "[d15678]",                  // BRAILLE PATTERN DOTS-15678
	0x28F2: "[d25678]",                  // BRAILLE PATTERN DOTS-25678
	0x28F3: "[d125678]",                 // BRAILLE PATTERN DOTS-125678
	0x28F4: "[d35678]",                  // BRAILLE PATTERN DOTS-35678
	0x28F5: "[d135678]",                 // BRAILLE PATTERN DOTS-135678
	0x28F6: "[d235678]",                 // BRAILLE PATTERN DOTS-235678
	0x28F7: "[d1235678]",                // BRAILLE PATTERN DOTS-1235678
	0x28F8: "[d45678]",                  // BRAILLE PATTERN DOTS-45678
	0x28F9: "[d145678]",                 // BRAILLE PATTERN DOTS-145678
	0x28FA: "[d245678]",                 // BRAILLE PATTERN DOTS-245678
	0x28FB: "[d1245678]",                // BRAILLE PATTERN DOTS-1245678
	0x28FC: "[d345678]",                 // BRAILLE PATTERN DOTS-345678
	0x28FD: "[d1345678]",                // BRAILLE PATTERN DOTS-1345678
	0x28FE: "[d2345678]",                // BRAILLE PATTERN DOTS-2345678
	0x28FF: "[d12345678]",               // BRAILLE PATTERN DOTS-12345678
	0x2983: "{",                         //
	0x2984: "} ",                        //
	0x2A74: "::=",                       //
	0x2A75: "==",                        //
	0x2A76: "===",                       //
	0x2A7D: "<=",                        //
	0x2A7E: ">=",                        //
	0x2C60: "L",                         //
	0x2C61: "l",                         //
	0x2C62: "L",                         //
	0x2C63: "P",                         //
	0x2C64: "R",                         //
	0x2C65: "a",                         //
	0x2C66: "t",                         //
	0x2C67: "H",                         //
	0x2C68: "h",                         //
	0x2C69: "K",                         //
	0x2C6A: "k",                         //
	0x2C6B: "Z",                         //
	0x2C6C: "z",                         //
	0x2C6D: "",                          //
	0x2C6E: "M",                         //
	0x2C6F: "A",                         //
	0x3001: ", ",                        // IDEOGRAPHIC COMMA
	0x3002: ". ",                        // IDEOGRAPHIC FULL STOP
	0x3003: "\"",                        // DITTO MARK
	0x3004: "[JIS]",                     // JAPANESE INDUSTRIAL STANDARD SYMBOL
	0x3005: "\"",                        // IDEOGRAPHIC ITERATION MARK
	0x3006: "/",                         // IDEOGRAPHIC CLOSING MARK
	0x3007: "0",                         // IDEOGRAPHIC NUMBER ZERO
	0x3008: "<",                         // LEFT ANGLE BRACKET
	0x3009: "> ",                        // RIGHT ANGLE BRACKET
	0x300A: "<<",                        // LEFT DOUBLE ANGLE BRACKET
	0x300B: ">> ",                       // RIGHT DOUBLE ANGLE BRACKET
	0x300C: "[",                         // LEFT CORNER BRACKET
	0x300D: "] ",                        // RIGHT CORNER BRACKET
	0x300E: "{",                         // LEFT WHITE CORNER BRACKET
	0x300F: "} ",                        // RIGHT WHITE CORNER BRACKET
	0x3010: "[(",                        // LEFT BLACK LENTICULAR BRACKET
	0x3011: ")] ",                       // RIGHT BLACK LENTICULAR BRACKET
	0x3012: "@",                         // POSTAL MARK
	0x3013: "X ",                        // GETA MARK
	0x3014: "[",                         // LEFT TORTOISE SHELL BRACKET
	0x3015: "] ",                        // RIGHT TORTOISE SHELL BRACKET
	0x3016: "[[",                        // LEFT WHITE LENTICULAR BRACKET
	0x3017: "]] ",                       // RIGHT WHITE LENTICULAR BRACKET
	0x3018: "((",                        // LEFT WHITE TORTOISE SHELL BRACKET
	0x3019: ")) ",                       // RIGHT WHITE TORTOISE SHELL BRACKET
	0x301A: "[[",                        // LEFT WHITE SQUARE BRACKET
	0x301B: "]] ",                       // RIGHT WHITE SQUARE BRACKET
	0x301C: "~ ",                        // WAVE DASH
	0x301D: "``",                        // REVERSED DOUBLE PRIME QUOTATION MARK
	0x301E: "''",                        // DOUBLE PRIME QUOTATION MARK
	0x301F: ",,",                        // LOW DOUBLE PRIME QUOTATION MARK
	0x3020: "@",                         // POSTAL MARK FACE
	0x3021: "1",                         // HANGZHOU NUMERAL ONE
	0x3022: "2",                         // HANGZHOU NUMERAL TWO
	0x3023: "3",                         // HANGZHOU NUMERAL THREE
	0x3024: "4",                         // HANGZHOU NUMERAL FOUR
	0x3025: "5",                         // HANGZHOU NUMERAL FIVE
	0x3026: "6",                         // HANGZHOU NUMERAL SIX
	0x3027: "7",                         // HANGZHOU NUMERAL SEVEN
	0x3028: "8",                         // HANGZHOU NUMERAL EIGHT
	0x3029: "9",                         // HANGZHOU NUMERAL NINE
	0x3030: "~",                         // WAVY DASH
	0x3031: "+",                         // VERTICAL KANA REPEAT MARK
	0x3032: "+",                         // VERTICAL KANA REPEAT WITH VOICED SOUND MARK
	0x3033: "+",                         // VERTICAL KANA REPEAT MARK UPPER HALF
	0x3034: "+",                         // VERTICAL KANA REPEAT WITH VOICED SOUND MARK UPPER H
	0x3035: "",                          // VERTICAL KANA REPEAT MARK LOWER HALF
	0x3036: "@",                         // CIRCLED POSTAL MARK
	0x3037: " // ",                      // IDEOGRAPHIC TELEGRAPH LINE FEED SEPARATOR SYMBOL
	0x3038: "+10+",                      // HANGZHOU NUMERAL TEN
	0x3039: "+20+",                      // HANGZHOU NUMERAL TWENTY
	0x303A: "+30+",                      // HANGZHOU NUMERAL THIRTY
}

// asciiRunes is merged from several sources

var asciiRunes = map[rune]string{
	0x0000: "",            // NULL
	0x0001: "",            // START OF HEADING
	0x0002: "",            // START OF TEXT
	0x0003: "",            // END OF TEXT
	0x0004: "",            // END OF TRANSMISSION
	0x0005: "",            // ENQUIRY
	0x0006: "",            // ACKNOWLEDGE
	0x0007: "",            // BELL
	0x0008: "",            // BACKSPACE
	0x0009: "",            // HORIZONTAL TABULATION
	0x000A: "",            // LINE FEED
	0x000B: "",            // VERTICAL TABULATION
	0x000C: "",            // FORM FEED
	0x000D: "",            // CARRIAGE RETURN
	0x000E: "",            // SHIFT OUT
	0x000F: "",            // SHIFT IN
	0x0010: "",            // DATA LINK ESCAPE
	0x0011: "",            // DEVICE CONTROL ONE
	0x0012: "",            // DEVICE CONTROL TWO
	0x0013: "",            // DEVICE CONTROL THREE
	0x0014: "",            // DEVICE CONTROL FOUR
	0x0015: "",            // NEGATIVE ACKNOWLEDGE
	0x0016: "",            // SYNCHRONOUS IDLE
	0x0017: "",            // END OF TRANSMISSION BLOCK
	0x0018: "",            // CANCEL
	0x0019: "",            // END OF MEDIUM
	0x001A: "",            // SUBSTITUTE
	0x001B: "",            // ESCAPE
	0x001C: "",            // FILE SEPARATOR
	0x001D: "",            // GROUP SEPARATOR
	0x001E: "",            // RECORD SEPARATOR
	0x001F: "",            // UNIT SEPARATOR
	0x0020: " ",           // SPACE
	0x0021: "!",           // EXCLAMATION MARK
	0x0022: "\"",          // QUOTATION MARK
	0x0023: "#",           // NUMBER SIGN
	0x0024: "$",           // DOLLAR SIGN
	0x0025: "%",           // PERCENT SIGN
	0x0026: "&",           // AMPERSAND
	0x0027: "'",           // APOSTROPHE
	0x0028: "(",           // LEFT PARENTHESIS
	0x0029: ")",           // RIGHT PARENTHESIS
	0x002A: "*",           // ASTERISK
	0x002B: "+",           // PLUS SIGN
	0x002C: ",",           // COMMA
	0x002D: "-",           // HYPHEN-MINUS
	0x002E: ".",           // FULL STOP
	0x002F: "/",           // SOLIDUS
	0x0030: "0",           // DIGIT ZERO
	0x0031: "1",           // DIGIT ONE
	0x0032: "2",           // DIGIT TWO
	0x0033: "3",           // DIGIT THREE
	0x0034: "4",           // DIGIT FOUR
	0x0035: "5",           // DIGIT FIVE
	0x0036: "6",           // DIGIT SIX
	0x0037: "7",           // DIGIT SEVEN
	0x0038: "8",           // DIGIT EIGHT
	0x0039: "9",           // DIGIT NINE
	0x003A: ":",           // COLON
	0x003B: ";",           // SEMICOLON
	0x003C: "<",           // LESS-THAN SIGN
	0x003D: "=",           // EQUALS SIGN
	0x003E: ">",           // GREATER-THAN SIGN
	0x003F: "?",           // QUESTION MARK
	0x0040: "@",           // COMMERCIAL AT
	0x0041: "A",           // LATIN CAPITAL LETTER A
	0x0042: "B",           // LATIN CAPITAL LETTER B
	0x0043: "C",           // LATIN CAPITAL LETTER C
	0x0044: "D",           // LATIN CAPITAL LETTER D
	0x0045: "E",           // LATIN CAPITAL LETTER E
	0x0046: "F",           // LATIN CAPITAL LETTER F
	0x0047: "G",           // LATIN CAPITAL LETTER G
	0x0048: "H",           // LATIN CAPITAL LETTER H
	0x0049: "I",           // LATIN CAPITAL LETTER I
	0x004A: "J",           // LATIN CAPITAL LETTER J
	0x004B: "K",           // LATIN CAPITAL LETTER K
	0x004C: "L",           // LATIN CAPITAL LETTER L
	0x004D: "M",           // LATIN CAPITAL LETTER M
	0x004E: "N",           // LATIN CAPITAL LETTER N
	0x004F: "O",           // LATIN CAPITAL LETTER O
	0x0050: "P",           // LATIN CAPITAL LETTER P
	0x0051: "Q",           // LATIN CAPITAL LETTER Q
	0x0052: "R",           // LATIN CAPITAL LETTER R
	0x0053: "S",           // LATIN CAPITAL LETTER S
	0x0054: "T",           // LATIN CAPITAL LETTER T
	0x0055: "U",           // LATIN CAPITAL LETTER U
	0x0056: "V",           // LATIN CAPITAL LETTER V
	0x0057: "W",           // LATIN CAPITAL LETTER W
	0x0058: "X",           // LATIN CAPITAL LETTER X
	0x0059: "Y",           // LATIN CAPITAL LETTER Y
	0x005A: "Z",           // LATIN CAPITAL LETTER Z
	0x005B: "[",           // LEFT SQUARE BRACKET
	0x005C: "\\",          // REVERSE SOLIDUS
	0x005D: "]",           // RIGHT SQUARE BRACKET
	0x005E: "^",           // CIRCUMFLEX ACCENT
	0x005F: "_",           // LOW LINE
	0x0060: "`",           // GRAVE ACCENT
	0x0061: "a",           // LATIN SMALL LETTER A
	0x0062: "b",           // LATIN SMALL LETTER B
	0x0063: "c",           // LATIN SMALL LETTER C
	0x0064: "d",           // LATIN SMALL LETTER D
	0x0065: "e",           // LATIN SMALL LETTER E
	0x0066: "f",           // LATIN SMALL LETTER F
	0x0067: "g",           // LATIN SMALL LETTER G
	0x0068: "h",           // LATIN SMALL LETTER H
	0x0069: "i",           // LATIN SMALL LETTER I
	0x006A: "j",           // LATIN SMALL LETTER J
	0x006B: "k",           // LATIN SMALL LETTER K
	0x006C: "l",           // LATIN SMALL LETTER L
	0x006D: "m",           // LATIN SMALL LETTER M
	0x006E: "n",           // LATIN SMALL LETTER N
	0x006F: "o",           // LATIN SMALL LETTER O
	0x0070: "p",           // LATIN SMALL LETTER P
	0x0071: "q",           // LATIN SMALL LETTER Q
	0x0072: "r",           // LATIN SMALL LETTER R
	0x0073: "s",           // LATIN SMALL LETTER S
	0x0074: "t",           // LATIN SMALL LETTER T
	0x0075: "u",           // LATIN SMALL LETTER U
	0x0076: "v",           // LATIN SMALL LETTER V
	0x0077: "w",           // LATIN SMALL LETTER W
	0x0078: "x",           // LATIN SMALL LETTER X
	0x0079: "y",           // LATIN SMALL LETTER Y
	0x007A: "z",           // LATIN SMALL LETTER Z
	0x007B: "{",           // LEFT CURLY BRACKET
	0x007C: "|",           // VERTICAL LINE
	0x007D: "}",           // RIGHT CURLY BRACKET
	0x007E: "~",           // TILDE
	0x007F: "",            // DELETE
	0x00A0: " ",           // NO-BREAK SPACE
	0x00A1: "!",           // INVERTED EXCLAMATION MARK
	0x00A2: " (cent) ",    // CENT SIGN
	0x00A3: " (pound) ",   // POUND SIGN
	0x00A4: " (curren) ",  // CURRENCY SIGN
	0x00A5: " (yen) ",     // YEN SIGN
	0x00A6: "|",           // BROKEN BAR
	0x00A7: " SS ",        // SECTION SIGN
	0x00A8: "",            // DIAERESIS
	0x00A9: " (copy) ",    // COPYRIGHT SIGN
	0x00AA: "",            // FEMININE ORDINAL INDICATOR
	0x00AB: "<<",          // LEFT-POINTING DOUBLE ANGLE QUOTATION MAR
	0x00AC: "!",           // NOT SIGN
	0x00AD: "-",           // SOFT HYPHEN
	0x00AE: " (reg) ",     // REGISTERED SIGN
	0x00AF: "-",           // MACRON
	0x00B0: " degrees ",   // DEGREE SIGN
	0x00B1: "+/-",         // PLUS-MINUS SIGN
	0x00B2: "(2)",         // SUPERSCRIPT TWO
	0x00B3: "(3)",         // SUPERSCRIPT THREE
	0x00B4: "'",           // ACUTE ACCENT
	0x00B5: " micro ",     // MICRO SIGN
	0x00B6: " (p) ",       // PILCROW SIGN
	0x00B7: ".",           // MIDDLE DOT
	0x00B8: "",            // CEDILLA
	0x00B9: "(1)",         // SUPERSCRIPT ONE
	0x00BA: "",            // MASCULINE ORDINAL INDICATOR
	0x00BB: ">>",          // RIGHT-POINTING DOUBLE ANGLE QUOTATION MA
	0x00BC: "(1/4)",       // VULGAR FRACTION ONE QUARTER
	0x00BD: "(1/2)",       // VULGAR FRACTION ONE HALF
	0x00BE: "(3/4)",       // VULGAR FRACTION THREE QUARTERS
	0x00BF: "?",           // INVERTED QUESTION MARK
	0x00C0: "A",           // LATIN CAPITAL LETTER A WITH GRAVE
	0x00C1: "A",           // LATIN CAPITAL LETTER A WITH ACUTE
	0x00C2: "A",           // LATIN CAPITAL LETTER A WITH CIRCUMFLEX
	0x00C3: "A",           // LATIN CAPITAL LETTER A WITH TILDE
	0x00C4: "A",           // LATIN CAPITAL LETTER A WITH DIAERESIS
	0x00C5: "A",           // LATIN CAPITAL LETTER A WITH RING ABOVE
	0x00C6: "AE",          // LATIN CAPITAL LETTER AE
	0x00C7: "C",           // LATIN CAPITAL LETTER C WITH CEDILLA
	0x00C8: "E",           // LATIN CAPITAL LETTER E WITH GRAVE
	0x00C9: "E",           // LATIN CAPITAL LETTER E WITH ACUTE
	0x00CA: "E",           // LATIN CAPITAL LETTER E WITH CIRCUMFLEX
	0x00CB: "E",           // LATIN CAPITAL LETTER E WITH DIAERESIS
	0x00CC: "I",           // LATIN CAPITAL LETTER I WITH GRAVE
	0x00CD: "I",           // LATIN CAPITAL LETTER I WITH ACUTE
	0x00CE: "I",           // LATIN CAPITAL LETTER I WITH CIRCUMFLEX
	0x00CF: "I",           // LATIN CAPITAL LETTER I WITH DIAERESIS
	0x00D0: "D",           // LATIN CAPITAL LETTER ETH
	0x00D1: "N",           // LATIN CAPITAL LETTER N WITH TILDE
	0x00D2: "O",           // LATIN CAPITAL LETTER O WITH GRAVE
	0x00D3: "O",           // LATIN CAPITAL LETTER O WITH ACUTE
	0x00D4: "O",           // LATIN CAPITAL LETTER O WITH CIRCUMFLEX
	0x00D5: "O",           // LATIN CAPITAL LETTER O WITH TILDE
	0x00D6: "O",           // LATIN CAPITAL LETTER O WITH DIAERESIS
	0x00D7: "x",           // MULTIPLICATION SIGN
	0x00D8: "O",           // LATIN CAPITAL LETTER O WITH STROKE
	0x00D9: "U",           // LATIN CAPITAL LETTER U WITH GRAVE
	0x00DA: "U",           // LATIN CAPITAL LETTER U WITH ACUTE
	0x00DB: "U",           // LATIN CAPITAL LETTER U WITH CIRCUMFLEX
	0x00DC: "U",           // LATIN CAPITAL LETTER U WITH DIAERESIS
	0x00DD: "Y",           // LATIN CAPITAL LETTER Y WITH ACUTE
	0x00DE: "Th",          // LATIN CAPITAL LETTER THORN
	0x00DF: "ss",          // LATIN SMALL LETTER SHARP S
	0x00E0: "a",           // LATIN SMALL LETTER A WITH GRAVE
	0x00E1: "a",           // LATIN SMALL LETTER A WITH ACUTE
	0x00E2: "a",           // LATIN SMALL LETTER A WITH CIRCUMFLEX
	0x00E3: "a",           // LATIN SMALL LETTER A WITH TILDE
	0x00E4: "a",           // LATIN SMALL LETTER A WITH DIAERESIS
	0x00E5: "a",           // LATIN SMALL LETTER A WITH RING ABOVE
	0x00E6: "ae",          // LATIN SMALL LETTER AE
	0x00E7: "c",           // LATIN SMALL LETTER C WITH CEDILLA
	0x00E8: "e",           // LATIN SMALL LETTER E WITH GRAVE
	0x00E9: "e",           // LATIN SMALL LETTER E WITH ACUTE
	0x00EA: "e",           // LATIN SMALL LETTER E WITH CIRCUMFLEX
	0x00EB: "e",           // LATIN SMALL LETTER E WITH DIAERESIS
	0x00EC: "i",           // LATIN SMALL LETTER I WITH GRAVE
	0x00ED: "i",           // LATIN SMALL LETTER I WITH ACUTE
	0x00EE: "i",           // LATIN SMALL LETTER I WITH CIRCUMFLEX
	0x00EF: "i",           // LATIN SMALL LETTER I WITH DIAERESIS
	0x00F0: "d",           // LATIN SMALL LETTER ETH
	0x00F1: "n",           // LATIN SMALL LETTER N WITH TILDE
	0x00F2: "o",           // LATIN SMALL LETTER O WITH GRAVE
	0x00F3: "o",           // LATIN SMALL LETTER O WITH ACUTE
	0x00F4: "o",           // LATIN SMALL LETTER O WITH CIRCUMFLEX
	0x00F5: "o",           // LATIN SMALL LETTER O WITH TILDE
	0x00F6: "o",           // LATIN SMALL LETTER O WITH DIAERESIS
	0x00F7: "/",           // DIVISION SIGN
	0x00F8: "o",           // LATIN SMALL LETTER O WITH STROKE
	0x00F9: "u",           // LATIN SMALL LETTER U WITH GRAVE
	0x00FA: "u",           // LATIN SMALL LETTER U WITH ACUTE
	0x00FB: "u",           // LATIN SMALL LETTER U WITH CIRCUMFLEX
	0x00FC: "u",           // LATIN SMALL LETTER U WITH DIAERESIS
	0x00FD: "y",           // LATIN SMALL LETTER Y WITH ACUTE
	0x00FE: "th",          // LATIN SMALL LETTER THORN
	0x00FF: "y",           // LATIN SMALL LETTER Y WITH DIAERESIS
	0x0100: "A",           // LATIN CAPITAL LETTER A WITH MACRON
	0x0101: "a",           // LATIN SMALL LETTER A WITH MACRON
	0x0102: "A",           // LATIN CAPITAL LETTER A WITH BREVE
	0x0103: "a",           // LATIN SMALL LETTER A WITH BREVE
	0x0104: "A",           // LATIN CAPITAL LETTER A WITH OGONEK
	0x0105: "a",           // LATIN SMALL LETTER A WITH OGONEK
	0x0106: "C",           // LATIN CAPITAL LETTER C WITH ACUTE
	0x0107: "c",           // LATIN SMALL LETTER C WITH ACUTE
	0x0108: "C",           // LATIN CAPITAL LETTER C WITH CIRCUMFLEX
	0x0109: "c",           // LATIN SMALL LETTER C WITH CIRCUMFLEX
	0x010A: "C",           // LATIN CAPITAL LETTER C WITH DOT ABOVE
	0x010B: "c",           // LATIN SMALL LETTER C WITH DOT ABOVE
	0x010C: "C",           // LATIN CAPITAL LETTER C WITH CARON
	0x010D: "c",           // LATIN SMALL LETTER C WITH CARON
	0x010E: "D",           // LATIN CAPITAL LETTER D WITH CARON
	0x010F: "d",           // LATIN SMALL LETTER D WITH CARON
	0x0110: "D",           // LATIN CAPITAL LETTER D WITH STROKE
	0x0111: "d",           // LATIN SMALL LETTER D WITH STROKE
	0x0112: "E",           // LATIN CAPITAL LETTER E WITH MACRON
	0x0113: "e",           // LATIN SMALL LETTER E WITH MACRON
	0x0114: "E",           // LATIN CAPITAL LETTER E WITH BREVE
	0x0115: "e",           // LATIN SMALL LETTER E WITH BREVE
	0x0116: "E",           // LATIN CAPITAL LETTER E WITH DOT ABOVE
	0x0117: "e",           // LATIN SMALL LETTER E WITH DOT ABOVE
	0x0118: "E",           // LATIN CAPITAL LETTER E WITH OGONEK
	0x0119: "e",           // LATIN SMALL LETTER E WITH OGONEK
	0x011A: "E",           // LATIN CAPITAL LETTER E WITH CARON
	0x011B: "e",           // LATIN SMALL LETTER E WITH CARON
	0x011C: "G",           // LATIN CAPITAL LETTER G WITH CIRCUMFLEX
	0x011D: "g",           // LATIN SMALL LETTER G WITH CIRCUMFLEX
	0x011E: "G",           // LATIN CAPITAL LETTER G WITH BREVE
	0x011F: "g",           // LATIN SMALL LETTER G WITH BREVE
	0x0120: "G",           // LATIN CAPITAL LETTER G WITH DOT ABOVE
	0x0121: "g",           // LATIN SMALL LETTER G WITH DOT ABOVE
	0x0122: "G",           // LATIN CAPITAL LETTER G WITH CEDILLA
	0x0123: "g",           // LATIN SMALL LETTER G WITH CEDILLA
	0x0124: "H",           // LATIN CAPITAL LETTER H WITH CIRCUMFLEX
	0x0125: "h",           // LATIN SMALL LETTER H WITH CIRCUMFLEX
	0x0126: "H",           // LATIN CAPITAL LETTER H WITH STROKE
	0x0127: "h",           // LATIN SMALL LETTER H WITH STROKE
	0x0128: "I",           // LATIN CAPITAL LETTER I WITH TILDE
	0x0129: "i",           // LATIN SMALL LETTER I WITH TILDE
	0x012A: "I",           // LATIN CAPITAL LETTER I WITH MACRON
	0x012B: "i",           // LATIN SMALL LETTER I WITH MACRON
	0x012C: "I",           // LATIN CAPITAL LETTER I WITH BREVE
	0x012D: "i",           // LATIN SMALL LETTER I WITH BREVE
	0x012E: "I",           // LATIN CAPITAL LETTER I WITH OGONEK
	0x012F: "i",           // LATIN SMALL LETTER I WITH OGONEK
	0x0130: "I",           // LATIN CAPITAL LETTER I WITH DOT ABOVE
	0x0131: "i",           // LATIN SMALL LETTER DOTLESS I
	0x0132: "IJ",          // LATIN CAPITAL LIGATURE IJ
	0x0133: "ij",          // LATIN SMALL LIGATURE IJ
	0x0134: "J",           // LATIN CAPITAL LETTER J WITH CIRCUMFLEX
	0x0135: "j",           // LATIN SMALL LETTER J WITH CIRCUMFLEX
	0x0136: "K",           // LATIN CAPITAL LETTER K WITH CEDILLA
	0x0137: "k",           // LATIN SMALL LETTER K WITH CEDILLA
	0x0138: "k",           // LATIN SMALL LETTER KRA
	0x0139: "L",           // LATIN CAPITAL LETTER L WITH ACUTE
	0x013A: "l",           // LATIN SMALL LETTER L WITH ACUTE
	0x013B: "L",           // LATIN CAPITAL LETTER L WITH CEDILLA
	0x013C: "l",           // LATIN SMALL LETTER L WITH CEDILLA
	0x013D: "L",           // LATIN CAPITAL LETTER L WITH CARON
	0x013E: "l",           // LATIN SMALL LETTER L WITH CARON
	0x013F: "L",           // LATIN CAPITAL LETTER L WITH MIDDLE DOT
	0x0140: "l",           // LATIN SMALL LETTER L WITH MIDDLE DOT
	0x0141: "L",           // LATIN CAPITAL LETTER L WITH STROKE
	0x0142: "l",           // LATIN SMALL LETTER L WITH STROKE
	0x0143: "N",           // LATIN CAPITAL LETTER N WITH ACUTE
	0x0144: "n",           // LATIN SMALL LETTER N WITH ACUTE
	0x0145: "N",           // LATIN CAPITAL LETTER N WITH CEDILLA
	0x0146: "n",           // LATIN SMALL LETTER N WITH CEDILLA
	0x0147: "N",           // LATIN CAPITAL LETTER N WITH CARON
	0x0148: "n",           // LATIN SMALL LETTER N WITH CARON
	0x0149: "'n",          // LATIN SMALL LETTER N PRECEDED BY APOSTROPHE
	0x014A: "NG",          // LATIN CAPITAL LETTER ENG
	0x014B: "ng",          // LATIN SMALL LETTER ENG
	0x014C: "O",           // LATIN CAPITAL LETTER O WITH MACRON
	0x014D: "o",           // LATIN SMALL LETTER O WITH MACRON
	0x014E: "O",           // LATIN CAPITAL LETTER O WITH BREVE
	0x014F: "o",           // LATIN SMALL LETTER O WITH BREVE
	0x0150: "O",           // LATIN CAPITAL LETTER O WITH DOUBLE ACUTE
	0x0151: "o",           // LATIN SMALL LETTER O WITH DOUBLE ACUTE
	0x0152: "OE",          // LATIN CAPITAL LIGATURE OE
	0x0153: "oe",          // LATIN SMALL LIGATURE OE
	0x0154: "R",           // LATIN CAPITAL LETTER R WITH ACUTE
	0x0155: "r",           // LATIN SMALL LETTER R WITH ACUTE
	0x0156: "R",           // LATIN CAPITAL LETTER R WITH CEDILLA
	0x0157: "r",           // LATIN SMALL LETTER R WITH CEDILLA
	0x0158: "R",           // LATIN CAPITAL LETTER R WITH CARON
	0x0159: "r",           // LATIN SMALL LETTER R WITH CARON
	0x015A: "S",           // LATIN CAPITAL LETTER S WITH ACUTE
	0x015B: "s",           // LATIN SMALL LETTER S WITH ACUTE
	0x015C: "S",           // LATIN CAPITAL LETTER S WITH CIRCUMFLEX
	0x015D: "s",           // LATIN SMALL LETTER S WITH CIRCUMFLEX
	0x015E: "S",           // LATIN CAPITAL LETTER S WITH CEDILLA
	0x015F: "s",           // LATIN SMALL LETTER S WITH CEDILLA
	0x0160: "S",           // LATIN CAPITAL LETTER S WITH CARON
	0x0161: "s",           // LATIN SMALL LETTER S WITH CARON
	0x0162: "T",           // LATIN CAPITAL LETTER T WITH CEDILLA
	0x0163: "t",           // LATIN SMALL LETTER T WITH CEDILLA
	0x0164: "T",           // LATIN CAPITAL LETTER T WITH CARON
	0x0165: "t",           // LATIN SMALL LETTER T WITH CARON
	0x0166: "T",           // LATIN CAPITAL LETTER T WITH STROKE
	0x0167: "t",           // LATIN SMALL LETTER T WITH STROKE
	0x0168: "U",           // LATIN CAPITAL LETTER U WITH TILDE
	0x0169: "u",           // LATIN SMALL LETTER U WITH TILDE
	0x016A: "U",           // LATIN CAPITAL LETTER U WITH MACRON
	0x016B: "u",           // LATIN SMALL LETTER U WITH MACRON
	0x016C: "U",           // LATIN CAPITAL LETTER U WITH BREVE
	0x016D: "u",           // LATIN SMALL LETTER U WITH BREVE
	0x016E: "U",           // LATIN CAPITAL LETTER U WITH RING ABOVE
	0x016F: "u",           // LATIN SMALL LETTER U WITH RING ABOVE
	0x0170: "U",           // LATIN CAPITAL LETTER U WITH DOUBLE ACUTE
	0x0171: "u",           // LATIN SMALL LETTER U WITH DOUBLE ACUTE
	0x0172: "U",           // LATIN CAPITAL LETTER U WITH OGONEK
	0x0173: "u",           // LATIN SMALL LETTER U WITH OGONEK
	0x0174: "W",           // LATIN CAPITAL LETTER W WITH CIRCUMFLEX
	0x0175: "w",           // LATIN SMALL LETTER W WITH CIRCUMFLEX
	0x0176: "Y",           // LATIN CAPITAL LETTER Y WITH CIRCUMFLEX
	0x0177: "y",           // LATIN SMALL LETTER Y WITH CIRCUMFLEX
	0x0178: "Y",           // LATIN CAPITAL LETTER Y WITH DIAERESIS
	0x0179: "Z",           // LATIN CAPITAL LETTER Z WITH ACUTE
	0x017A: "z",           // LATIN SMALL LETTER Z WITH ACUTE
	0x017B: "Z",           // LATIN CAPITAL LETTER Z WITH DOT ABOVE
	0x017C: "z",           // LATIN SMALL LETTER Z WITH DOT ABOVE
	0x017D: "Z",           // LATIN CAPITAL LETTER Z WITH CARON
	0x017E: "z",           // LATIN SMALL LETTER Z WITH CARON
	0x017F: "s",           // LATIN SMALL LETTER LONG S
	0x0180: "b",           // LATIN SMALL LETTER B WITH STROKE
	0x0181: "B",           // LATIN CAPITAL LETTER B WITH HOOK
	0x0182: "B",           // LATIN CAPITAL LETTER B WITH TOPBAR
	0x0183: "b",           // LATIN SMALL LETTER B WITH TOPBAR
	0x0184: "6",           // LATIN CAPITAL LETTER TONE SIX
	0x0185: "6",           // LATIN SMALL LETTER TONE SIX
	0x0186: "O",           // LATIN CAPITAL LETTER OPEN O
	0x0187: "C",           // LATIN CAPITAL LETTER C WITH HOOK
	0x0188: "c",           // LATIN SMALL LETTER C WITH HOOK
	0x0189: "D",           // LATIN CAPITAL LETTER AFRICAN D
	0x018A: "D",           // LATIN CAPITAL LETTER D WITH HOOK
	0x018B: "D",           // LATIN CAPITAL LETTER D WITH TOPBAR
	0x018C: "d",           // LATIN SMALL LETTER D WITH TOPBAR
	0x018D: "d",           // LATIN SMALL LETTER TURNED DELTA
	0x018E: "3",           // LATIN CAPITAL LETTER REVERSED E
	0x018F: "@",           // LATIN CAPITAL LETTER SCHWA
	0x0190: "E",           // LATIN CAPITAL LETTER OPEN E
	0x0191: "F",           // LATIN CAPITAL LETTER F WITH HOOK
	0x0192: "f",           // LATIN SMALL LETTER F WITH HOOK
	0x0193: "G",           // LATIN CAPITAL LETTER G WITH HOOK
	0x0194: "G",           // LATIN CAPITAL LETTER GAMMA
	0x0195: "hv",          // LATIN SMALL LETTER HV
	0x0196: "I",           // LATIN CAPITAL LETTER IOTA
	0x0197: "I",           // LATIN CAPITAL LETTER I WITH STROKE
	0x0198: "K",           // LATIN CAPITAL LETTER K WITH HOOK
	0x0199: "k",           // LATIN SMALL LETTER K WITH HOOK
	0x019A: "l",           // LATIN SMALL LETTER L WITH BAR
	0x019B: "l",           // LATIN SMALL LETTER LAMBDA WITH STROKE
	0x019C: "W",           // LATIN CAPITAL LETTER TURNED M
	0x019D: "N",           // LATIN CAPITAL LETTER N WITH LEFT HOOK
	0x019E: "n",           // LATIN SMALL LETTER N WITH LONG RIGHT LEG
	0x019F: "O",           // LATIN CAPITAL LETTER O WITH MIDDLE TILDE
	0x01A0: "O",           // LATIN CAPITAL LETTER O WITH HORN
	0x01A1: "o",           // LATIN SMALL LETTER O WITH HORN
	0x01A2: "OI",          // LATIN CAPITAL LETTER OI
	0x01A3: "oi",          // LATIN SMALL LETTER OI
	0x01A4: "P",           // LATIN CAPITAL LETTER P WITH HOOK
	0x01A5: "p",           // LATIN SMALL LETTER P WITH HOOK
	0x01A6: "YR",          // LATIN LETTER YR
	0x01A7: "2",           // LATIN CAPITAL LETTER TONE TWO
	0x01A8: "2",           // LATIN SMALL LETTER TONE TWO
	0x01A9: "SH",          // LATIN CAPITAL LETTER ESH
	0x01AA: "sh",          // LATIN LETTER REVERSED ESH LOOP
	0x01AB: "t",           // LATIN SMALL LETTER T WITH PALATAL HOOK
	0x01AC: "T",           // LATIN CAPITAL LETTER T WITH HOOK
	0x01AD: "t",           // LATIN SMALL LETTER T WITH HOOK
	0x01AE: "T",           // LATIN CAPITAL LETTER T WITH RETROFLEX HOOK
	0x01AF: "U",           // LATIN CAPITAL LETTER U WITH HORN
	0x01B0: "u",           // LATIN SMALL LETTER U WITH HORN
	0x01B1: "Y",           // LATIN CAPITAL LETTER UPSILON
	0x01B2: "V",           // LATIN CAPITAL LETTER V WITH HOOK
	0x01B3: "Y",           // LATIN CAPITAL LETTER Y WITH HOOK
	0x01B4: "y",           // LATIN SMALL LETTER Y WITH HOOK
	0x01B5: "Z",           // LATIN CAPITAL LETTER Z WITH STROKE
	0x01B6: "z",           // LATIN SMALL LETTER Z WITH STROKE
	0x01B7: "ZH",          // LATIN CAPITAL LETTER EZH
	0x01B8: "ZH",          // LATIN CAPITAL LETTER EZH REVERSED
	0x01B9: "zh",          // LATIN SMALL LETTER EZH REVERSED
	0x01BA: "zh",          // LATIN SMALL LETTER EZH WITH TAIL
	0x01BB: "2",           // LATIN LETTER TWO WITH STROKE
	0x01BC: "5",           // LATIN CAPITAL LETTER TONE FIVE
	0x01BD: "5",           // LATIN SMALL LETTER TONE FIVE
	0x01BE: "ts",          // LATIN LETTER INVERTED GLOTTAL STOP WITH STROKE
	0x01BF: "w",           // LATIN LETTER WYNN
	0x01C0: "|",           // LATIN LETTER DENTAL CLICK
	0x01C1: "||",          // LATIN LETTER LATERAL CLICK
	0x01C2: "|=",          // LATIN LETTER ALVEOLAR CLICK
	0x01C3: "!",           // LATIN LETTER RETROFLEX CLICK
	0x01C4: "DZ",          // LATIN CAPITAL LETTER DZ WITH CARON
	0x01C5: "Dz",          // LATIN CAPITAL LETTER D WITH SMALL LETTER Z WITH CA
	0x01C6: "dz",          // LATIN SMALL LETTER DZ WITH CARON
	0x01C7: "LJ",          // LATIN CAPITAL LETTER LJ
	0x01C8: "Lj",          // LATIN CAPITAL LETTER L WITH SMALL LETTER J
	0x01C9: "lj",          // LATIN SMALL LETTER LJ
	0x01CA: "NJ",          // LATIN CAPITAL LETTER NJ
	0x01CB: "Nj",          // LATIN CAPITAL LETTER N WITH SMALL LETTER J
	0x01CC: "nj",          // LATIN SMALL LETTER NJ
	0x01CD: "A",           // LATIN CAPITAL LETTER A WITH CARON
	0x01CE: "a",           // LATIN SMALL LETTER A WITH CARON
	0x01CF: "I",           // LATIN CAPITAL LETTER I WITH CARON
	0x01D0: "i",           // LATIN SMALL LETTER I WITH CARON
	0x01D1: "O",           // LATIN CAPITAL LETTER O WITH CARON
	0x01D2: "o",           // LATIN SMALL LETTER O WITH CARON
	0x01D3: "U",           // LATIN CAPITAL LETTER U WITH CARON
	0x01D4: "u",           // LATIN SMALL LETTER U WITH CARON
	0x01D5: "U",           // LATIN CAPITAL LETTER U WITH DIAERESIS AND MACRON
	0x01D6: "u",           // LATIN SMALL LETTER U WITH DIAERESIS AND MACRON
	0x01D7: "U",           // LATIN CAPITAL LETTER U WITH DIAERESIS AND ACUTE
	0x01D8: "u",           // LATIN SMALL LETTER U WITH DIAERESIS AND ACUTE
	0x01D9: "U",           // LATIN CAPITAL LETTER U WITH DIAERESIS AND CARON
	0x01DA: "u",           // LATIN SMALL LETTER U WITH DIAERESIS AND CARON
	0x01DB: "U",           // LATIN CAPITAL LETTER U WITH DIAERESIS AND GRAVE
	0x01DC: "u",           // LATIN SMALL LETTER U WITH DIAERESIS AND GRAVE
	0x01DD: "@",           // LATIN SMALL LETTER TURNED E
	0x01DE: "A",           // LATIN CAPITAL LETTER A WITH DIAERESIS AND MACRON
	0x01DF: "a",           // LATIN SMALL LETTER A WITH DIAERESIS AND MACRON
	0x01E0: "A",           // LATIN CAPITAL LETTER A WITH DOT ABOVE AND MACRON
	0x01E1: "a",           // LATIN SMALL LETTER A WITH DOT ABOVE AND MACRON
	0x01E2: "AE",          // LATIN CAPITAL LETTER AE WITH MACRON
	0x01E3: "ae",          // LATIN SMALL LETTER AE WITH MACRON
	0x01E4: "G",           // LATIN CAPITAL LETTER G WITH STROKE
	0x01E5: "g",           // LATIN SMALL LETTER G WITH STROKE
	0x01E6: "G",           // LATIN CAPITAL LETTER G WITH CARON
	0x01E7: "g",           // LATIN SMALL LETTER G WITH CARON
	0x01E8: "K",           // LATIN CAPITAL LETTER K WITH CARON
	0x01E9: "k",           // LATIN SMALL LETTER K WITH CARON
	0x01EA: "O",           // LATIN CAPITAL LETTER O WITH OGONEK
	0x01EB: "o",           // LATIN SMALL LETTER O WITH OGONEK
	0x01EC: "O",           // LATIN CAPITAL LETTER O WITH OGONEK AND MACRON
	0x01ED: "o",           // LATIN SMALL LETTER O WITH OGONEK AND MACRON
	0x01EE: "ZH",          // LATIN CAPITAL LETTER EZH WITH CARON
	0x01EF: "zh",          // LATIN SMALL LETTER EZH WITH CARON
	0x01F0: "j",           // LATIN SMALL LETTER J WITH CARON
	0x01F1: "DZ",          // LATIN CAPITAL LETTER DZ
	0x01F2: "Dz",          // LATIN CAPITAL LETTER D WITH SMALL LETTER Z
	0x01F3: "dz",          // LATIN SMALL LETTER DZ
	0x01F4: "G",           // LATIN CAPITAL LETTER G WITH ACUTE
	0x01F5: "g",           // LATIN SMALL LETTER G WITH ACUTE
	0x01F6: "HV",          // LATIN CAPITAL LETTER HWAIR
	0x01F7: "W",           // LATIN CAPITAL LETTER WYNN
	0x01F8: "N",           // LATIN CAPITAL LETTER N WITH GRAVE
	0x01F9: "n",           // LATIN SMALL LETTER N WITH GRAVE
	0x01FA: "A",           // LATIN CAPITAL LETTER A WITH RING ABOVE AND ACUTE
	0x01FB: "a",           // LATIN SMALL LETTER A WITH RING ABOVE AND ACUTE
	0x01FC: "AE",          // LATIN CAPITAL LETTER AE WITH ACUTE
	0x01FD: "ae",          // LATIN SMALL LETTER AE WITH ACUTE
	0x01FE: "O",           // LATIN CAPITAL LETTER O WITH STROKE AND ACUTE
	0x01FF: "o",           // LATIN SMALL LETTER O WITH STROKE AND ACUTE
	0x0200: "A",           // LATIN CAPITAL LETTER A WITH DOUBLE GRAVE
	0x0201: "a",           // LATIN SMALL LETTER A WITH DOUBLE GRAVE
	0x0202: "A",           // LATIN CAPITAL LETTER A WITH INVERTED BREVE
	0x0203: "a",           // LATIN SMALL LETTER A WITH INVERTED BREVE
	0x0204: "E",           // LATIN CAPITAL LETTER E WITH DOUBLE GRAVE
	0x0205: "e",           // LATIN SMALL LETTER E WITH DOUBLE GRAVE
	0x0206: "E",           // LATIN CAPITAL LETTER E WITH INVERTED BREVE
	0x0207: "e",           // LATIN SMALL LETTER E WITH INVERTED BREVE
	0x0208: "I",           // LATIN CAPITAL LETTER I WITH DOUBLE GRAVE
	0x0209: "i",           // LATIN SMALL LETTER I WITH DOUBLE GRAVE
	0x020A: "I",           // LATIN CAPITAL LETTER I WITH INVERTED BREVE
	0x020B: "i",           // LATIN SMALL LETTER I WITH INVERTED BREVE
	0x020C: "O",           // LATIN CAPITAL LETTER O WITH DOUBLE GRAVE
	0x020D: "o",           // LATIN SMALL LETTER O WITH DOUBLE GRAVE
	0x020E: "O",           // LATIN CAPITAL LETTER O WITH INVERTED BREVE
	0x020F: "o",           // LATIN SMALL LETTER O WITH INVERTED BREVE
	0x0210: "R",           // LATIN CAPITAL LETTER R WITH DOUBLE GRAVE
	0x0211: "r",           // LATIN SMALL LETTER R WITH DOUBLE GRAVE
	0x0212: "R",           // LATIN CAPITAL LETTER R WITH INVERTED BREVE
	0x0213: "r",           // LATIN SMALL LETTER R WITH INVERTED BREVE
	0x0214: "U",           // LATIN CAPITAL LETTER U WITH DOUBLE GRAVE
	0x0215: "u",           // LATIN SMALL LETTER U WITH DOUBLE GRAVE
	0x0216: "U",           // LATIN CAPITAL LETTER U WITH INVERTED BREVE
	0x0217: "u",           // LATIN SMALL LETTER U WITH INVERTED BREVE
	0x0218: "S",           // LATIN CAPITAL LETTER S WITH COMMA BELOW
	0x0219: "s",           // LATIN SMALL LETTER S WITH COMMA BELOW
	0x021A: "T",           // LATIN CAPITAL LETTER T WITH COMMA BELOW
	0x021B: "t",           // LATIN SMALL LETTER T WITH COMMA BELOW
	0x021C: "Y",           // LATIN CAPITAL LETTER YOGH
	0x021D: "y",           // LATIN SMALL LETTER YOGH
	0x021E: "H",           // LATIN CAPITAL LETTER H WITH CARON
	0x021F: "h",           // LATIN SMALL LETTER H WITH CARON
	0x0220: "N",           //
	0x0221: "d",           //
	0x0222: "OU",          // LATIN CAPITAL LETTER OU
	0x0223: "ou",          // LATIN SMALL LETTER OU
	0x0224: "Z",           // LATIN CAPITAL LETTER Z WITH HOOK
	0x0225: "z",           // LATIN SMALL LETTER Z WITH HOOK
	0x0226: "A",           // LATIN CAPITAL LETTER A WITH DOT ABOVE
	0x0227: "a",           // LATIN SMALL LETTER A WITH DOT ABOVE
	0x0228: "E",           // LATIN CAPITAL LETTER E WITH CEDILLA
	0x0229: "e",           // LATIN SMALL LETTER E WITH CEDILLA
	0x022A: "O",           // LATIN CAPITAL LETTER O WITH DIAERESIS AND MACRON
	0x022B: "o",           // LATIN SMALL LETTER O WITH DIAERESIS AND MACRON
	0x022C: "O",           // LATIN CAPITAL LETTER O WITH TILDE AND MACRON
	0x022D: "o",           // LATIN SMALL LETTER O WITH TILDE AND MACRON
	0x022E: "O",           // LATIN CAPITAL LETTER O WITH DOT ABOVE
	0x022F: "o",           // LATIN SMALL LETTER O WITH DOT ABOVE
	0x0230: "O",           // LATIN CAPITAL LETTER O WITH DOT ABOVE AND MACRON
	0x0231: "o",           // LATIN SMALL LETTER O WITH DOT ABOVE AND MACRON
	0x0232: "Y",           // LATIN CAPITAL LETTER Y WITH MACRON
	0x0233: "y",           // LATIN SMALL LETTER Y WITH MACRON
	0x0234: "l",           //
	0x0235: "n",           //
	0x0236: "t",           //
	0x0237: "j",           //
	0x0238: "db",          //
	0x0239: "qp",          //
	0x023A: "A",           //
	0x023B: "C",           //
	0x023C: "c",           //
	0x023D: "L",           //
	0x023E: "T",           //
	0x023F: "s",           //
	0x0240: "z",           //
	0x0243: "B",           //
	0x0244: "U",           //
	0x0245: "^",           //
	0x0246: "E",           //
	0x0247: "e",           //
	0x0248: "J",           //
	0x0249: "j",           //
	0x024A: "q",           //
	0x024B: "q",           //
	0x024C: "R",           //
	0x024D: "r",           //
	0x024E: "Y",           //
	0x024F: "y",           //
	0x0250: "a",           // LATIN SMALL LETTER TURNED A
	0x0251: "a",           // LATIN SMALL LETTER ALPHA
	0x0252: "a",           // LATIN SMALL LETTER TURNED ALPHA
	0x0253: "b",           // LATIN SMALL LETTER B WITH HOOK
	0x0254: "o",           // LATIN SMALL LETTER OPEN O
	0x0255: "c",           // LATIN SMALL LETTER C WITH CURL
	0x0256: "d",           // LATIN SMALL LETTER D WITH TAIL
	0x0257: "d",           // LATIN SMALL LETTER D WITH HOOK
	0x0258: "e",           // LATIN SMALL LETTER REVERSED E
	0x0259: "@",           // LATIN SMALL LETTER SCHWA
	0x025A: "@",           // LATIN SMALL LETTER SCHWA WITH HOOK
	0x025B: "e",           // LATIN SMALL LETTER OPEN E
	0x025C: "e",           // LATIN SMALL LETTER REVERSED OPEN E
	0x025D: "e",           // LATIN SMALL LETTER REVERSED OPEN E WITH HOOK
	0x025E: "e",           // LATIN SMALL LETTER CLOSED REVERSED OPEN E
	0x025F: "j",           // LATIN SMALL LETTER DOTLESS J WITH STROKE
	0x0260: "g",           // LATIN SMALL LETTER G WITH HOOK
	0x0261: "g",           // LATIN SMALL LETTER SCRIPT G
	0x0262: "g",           // LATIN LETTER SMALL CAPITAL G
	0x0263: "g",           // LATIN SMALL LETTER GAMMA
	0x0264: "u",           // LATIN SMALL LETTER RAMS HORN
	0x0265: "Y",           // LATIN SMALL LETTER TURNED H
	0x0266: "h",           // LATIN SMALL LETTER H WITH HOOK
	0x0267: "h",           // LATIN SMALL LETTER HENG WITH HOOK
	0x0268: "i",           // LATIN SMALL LETTER I WITH STROKE
	0x0269: "i",           // LATIN SMALL LETTER IOTA
	0x026A: "I",           // LATIN LETTER SMALL CAPITAL I
	0x026B: "l",           // LATIN SMALL LETTER L WITH MIDDLE TILDE
	0x026C: "l",           // LATIN SMALL LETTER L WITH BELT
	0x026D: "l",           // LATIN SMALL LETTER L WITH RETROFLEX HOOK
	0x026E: "lZ",          // LATIN SMALL LETTER LEZH
	0x026F: "W",           // LATIN SMALL LETTER TURNED M
	0x0270: "W",           // LATIN SMALL LETTER TURNED M WITH LONG LEG
	0x0271: "m",           // LATIN SMALL LETTER M WITH HOOK
	0x0272: "n",           // LATIN SMALL LETTER N WITH LEFT HOOK
	0x0273: "n",           // LATIN SMALL LETTER N WITH RETROFLEX HOOK
	0x0274: "n",           // LATIN LETTER SMALL CAPITAL N
	0x0275: "o",           // LATIN SMALL LETTER BARRED O
	0x0276: "OE",          // LATIN LETTER SMALL CAPITAL OE
	0x0277: "O",           // LATIN SMALL LETTER CLOSED OMEGA
	0x0278: "F",           // LATIN SMALL LETTER PHI
	0x0279: "r",           // LATIN SMALL LETTER TURNED R
	0x027A: "r",           // LATIN SMALL LETTER TURNED R WITH LONG LEG
	0x027B: "r",           // LATIN SMALL LETTER TURNED R WITH HOOK
	0x027C: "r",           // LATIN SMALL LETTER R WITH LONG LEG
	0x027D: "r",           // LATIN SMALL LETTER R WITH TAIL
	0x027E: "r",           // LATIN SMALL LETTER R WITH FISHHOOK
	0x027F: "r",           // LATIN SMALL LETTER REVERSED R WITH FISHHOOK
	0x0280: "R",           // LATIN LETTER SMALL CAPITAL R
	0x0281: "R",           // LATIN LETTER SMALL CAPITAL INVERTED R
	0x0282: "s",           // LATIN SMALL LETTER S WITH HOOK
	0x0283: "S",           // LATIN SMALL LETTER ESH
	0x0284: "j",           // LATIN SMALL LETTER DOTLESS J WITH STROKE AND HOOK
	0x0285: "S",           // LATIN SMALL LETTER SQUAT REVERSED ESH
	0x0286: "S",           // LATIN SMALL LETTER ESH WITH CURL
	0x0287: "t",           // LATIN SMALL LETTER TURNED T
	0x0288: "t",           // LATIN SMALL LETTER T WITH RETROFLEX HOOK
	0x0289: "u",           // LATIN SMALL LETTER U BAR
	0x028A: "U",           // LATIN SMALL LETTER UPSILON
	0x028B: "v",           // LATIN SMALL LETTER V WITH HOOK
	0x028C: "^",           // LATIN SMALL LETTER TURNED V
	0x028D: "w",           // LATIN SMALL LETTER TURNED W
	0x028E: "y",           // LATIN SMALL LETTER TURNED Y
	0x028F: "Y",           // LATIN LETTER SMALL CAPITAL Y
	0x0290: "z",           // LATIN SMALL LETTER Z WITH RETROFLEX HOOK
	0x0291: "z",           // LATIN SMALL LETTER Z WITH CURL
	0x0292: "Z",           // LATIN SMALL LETTER EZH
	0x0293: "Z",           // LATIN SMALL LETTER EZH WITH CURL
	0x0297: "C",           // LATIN LETTER STRETCHED C
	0x0298: "@",           // LATIN LETTER BILABIAL CLICK
	0x0299: "B",           // LATIN LETTER SMALL CAPITAL B
	0x029A: "E",           // LATIN SMALL LETTER CLOSED OPEN E
	0x029B: "G",           // LATIN LETTER SMALL CAPITAL G WITH HOOK
	0x029C: "H",           // LATIN LETTER SMALL CAPITAL H
	0x029D: "j",           // LATIN SMALL LETTER J WITH CROSSED-TAIL
	0x029E: "k",           // LATIN SMALL LETTER TURNED K
	0x029F: "L",           // LATIN LETTER SMALL CAPITAL L
	0x02A0: "q",           // LATIN SMALL LETTER Q WITH HOOK
	0x02A3: "dz",          // LATIN SMALL LETTER DZ DIGRAPH
	0x02A4: "dZ",          // LATIN SMALL LETTER DEZH DIGRAPH
	0x02A5: "dz",          // LATIN SMALL LETTER DZ DIGRAPH WITH CURL
	0x02A6: "ts",          // LATIN SMALL LETTER TS DIGRAPH
	0x02A7: "tS",          // LATIN SMALL LETTER TESH DIGRAPH
	0x02A8: "tC",          // LATIN SMALL LETTER TC DIGRAPH WITH CURL
	0x02A9: "fN",          // LATIN SMALL LETTER FENG DIGRAPH
	0x02AA: "ls",          // LATIN SMALL LETTER LS DIGRAPH
	0x02AB: "lz",          // LATIN SMALL LETTER LZ DIGRAPH
	0x02AC: "WW",          // LATIN LETTER BILABIAL PERCUSSIVE
	0x02AD: "]]",          // LATIN LETTER BIDENTAL PERCUSSIVE
	0x02AE: "h",           //
	0x02AF: "h",           //
	0x02B0: "k",           // MODIFIER LETTER SMALL H
	0x02B1: "h",           // MODIFIER LETTER SMALL H WITH HOOK
	0x02B2: "j",           // MODIFIER LETTER SMALL J
	0x02B3: "r",           // MODIFIER LETTER SMALL R
	0x02B4: "r",           // MODIFIER LETTER SMALL TURNED R
	0x02B5: "r",           // MODIFIER LETTER SMALL TURNED R WITH HOOK
	0x02B6: "r",           // MODIFIER LETTER SMALL CAPITAL INVERTED R
	0x02B7: "w",           // MODIFIER LETTER SMALL W
	0x02B8: "y",           // MODIFIER LETTER SMALL Y
	0x02B9: "'",           // MODIFIER LETTER PRIME
	0x02BA: "\"",          // MODIFIER LETTER DOUBLE PRIME
	0x02BB: "`",           // MODIFIER LETTER TURNED COMMA
	0x02BC: "'",           // MODIFIER LETTER APOSTROPHE
	0x02BD: "`",           // MODIFIER LETTER REVERSED COMMA
	0x02BE: "`",           // MODIFIER LETTER RIGHT HALF RING
	0x02BF: "'",           // MODIFIER LETTER LEFT HALF RING
	0x02C0: "?",           // MODIFIER LETTER GLOTTAL STOP
	0x02C1: "?",           // MODIFIER LETTER REVERSED GLOTTAL STOP
	0x02C2: "<",           // MODIFIER LETTER LEFT ARROWHEAD
	0x02C3: ">",           // MODIFIER LETTER RIGHT ARROWHEAD
	0x02C4: "^",           // MODIFIER LETTER UP ARROWHEAD
	0x02C5: "v",           // MODIFIER LETTER DOWN ARROWHEAD
	0x02C6: "^",           // MODIFIER LETTER CIRCUMFLEX ACCENT
	0x02C7: "V",           // CARON
	0x02C8: "'",           // MODIFIER LETTER VERTICAL LINE
	0x02C9: "-",           // MODIFIER LETTER MACRON
	0x02CA: "'",           // MODIFIER LETTER ACUTE ACCENT
	0x02CB: "`",           // MODIFIER LETTER GRAVE ACCENT
	0x02CC: ",",           // MODIFIER LETTER LOW VERTICAL LINE
	0x02CD: "_",           // MODIFIER LETTER LOW MACRON
	0x02CE: "'",           // MODIFIER LETTER LOW GRAVE ACCENT
	0x02CF: "`",           // MODIFIER LETTER LOW ACUTE ACCENT
	0x02D0: ":",           // MODIFIER LETTER TRIANGULAR COLON
	0x02D1: ".",           // MODIFIER LETTER HALF TRIANGULAR COLON
	0x02D2: "`",           // MODIFIER LETTER CENTRED RIGHT HALF RING
	0x02D3: "'",           // MODIFIER LETTER CENTRED LEFT HALF RING
	0x02D4: "^",           // MODIFIER LETTER UP TACK
	0x02D5: "V",           // MODIFIER LETTER DOWN TACK
	0x02D6: "+",           // MODIFIER LETTER PLUS SIGN
	0x02D7: "-",           // MODIFIER LETTER MINUS SIGN
	0x02D8: "V",           // BREVE
	0x02D9: ".",           // DOT ABOVE
	0x02DA: "@",           // RING ABOVE
	0x02DB: ",",           // OGONEK
	0x02DC: "~",           // SMALL TILDE
	0x02DD: "",            // DOUBLE ACUTE ACCENT
	0x02DE: "R",           // MODIFIER LETTER RHOTIC HOOK
	0x02DF: "X",           // MODIFIER LETTER CROSS ACCENT
	0x02E0: "G",           // MODIFIER LETTER SMALL GAMMA
	0x02E1: "l",           // MODIFIER LETTER SMALL L
	0x02E2: "s",           // MODIFIER LETTER SMALL S
	0x02E3: "x",           // MODIFIER LETTER SMALL X
	0x02EC: "V",           // MODIFIER LETTER VOICING
	0x02ED: "=",           // MODIFIER LETTER UNASPIRATED
	0x02EE: "\"",          // MODIFIER LETTER DOUBLE APOSTROPHE
	0x02F1: "l",           //
	0x02F2: "s",           //
	0x02F3: "x",           //
	0x02FC: "v",           //
	0x02FD: "=",           //
	0x0363: "a",           //
	0x0364: "e",           //
	0x0365: "i",           //
	0x0366: "o",           //
	0x0367: "u",           //
	0x0368: "c",           //
	0x0369: "d",           //
	0x036A: "h",           //
	0x036B: "m",           //
	0x036C: "r",           //
	0x036D: "t",           //
	0x036E: "v",           //
	0x036F: "x",           //
	0x0374: "'",           // GREEK NUMERAL SIGN
	0x0375: ",",           // GREEK LOWER NUMERAL SIGN
	0x037E: "?",           // GREEK QUESTION MARK
	0x037F: "J",           //
	0x0386: "A",           // GREEK CAPITAL LETTER ALPHA WITH TONOS
	0x0387: "",            // GREEK ANO TELEIA
	0x0388: "E",           // GREEK CAPITAL LETTER EPSILON WITH TONOS
	0x0389: "E",           // GREEK CAPITAL LETTER ETA WITH TONOS
	0x038A: "I",           // GREEK CAPITAL LETTER IOTA WITH TONOS
	0x038C: "O",           // GREEK CAPITAL LETTER OMICRON WITH TONOS
	0x038E: "U",           // GREEK CAPITAL LETTER UPSILON WITH TONOS
	0x038F: " Omega ",     // GREEK CAPITAL LETTER OMEGA WITH TONOS
	0x0390: " iota ",      // GREEK SMALL LETTER IOTA WITH DIALYTIKA AND TONOS
	0x0391: " Alpha ",     // GREEK CAPITAL LETTER ALPHA
	0x0392: " Beta ",      // GREEK CAPITAL LETTER BETA
	0x0393: " Gamma ",     // GREEK CAPITAL LETTER GAMMA
	0x0394: " Delta ",     // GREEK CAPITAL LETTER DELTA
	0x0395: " Epsilon",    // GREEK CAPITAL LETTER EPSILON
	0x0396: " Zeta ",      // GREEK CAPITAL LETTER ZETA
	0x0397: " Eta ",       // GREEK CAPITAL LETTER ETA
	0x0398: " Theta ",     // GREEK CAPITAL LETTER THETA
	0x0399: " Iota ",      // GREEK CAPITAL LETTER IOTA
	0x039A: " Kappa ",     // GREEK CAPITAL LETTER KAPPA
	0x039B: " Lambda ",    // GREEK CAPITAL LETTER LAMDA
	0x039C: " Mu ",        // GREEK CAPITAL LETTER MU
	0x039D: " Nu ",        // GREEK CAPITAL LETTER NU
	0x039E: " Xi ",        // GREEK CAPITAL LETTER XI
	0x039F: " Omicron ",   // GREEK CAPITAL LETTER OMICRON
	0x03A0: " Pi ",        // GREEK CAPITAL LETTER PI
	0x03A1: " Rho ",       // GREEK CAPITAL LETTER RHO
	0x03A3: " Sigma ",     // GREEK CAPITAL LETTER SIGMA
	0x03A4: " Tau ",       // GREEK CAPITAL LETTER TAU
	0x03A5: " Upsilon ",   // GREEK CAPITAL LETTER UPSILON
	0x03A6: " Phi ",       // GREEK CAPITAL LETTER PHI
	0x03A7: " Chi ",       // GREEK CAPITAL LETTER CHI
	0x03A8: " Psi ",       // GREEK CAPITAL LETTER PSI
	0x03A9: " Omega ",     // GREEK CAPITAL LETTER OMEGA
	0x03AA: " Iota ",      // GREEK CAPITAL LETTER IOTA WITH DIALYTIKA
	0x03AB: " Upsilon ",   // GREEK CAPITAL LETTER UPSILON WITH DIALYTIKA
	0x03AC: " alpha ",     // GREEK SMALL LETTER ALPHA WITH TONOS
	0x03AD: " epsilon ",   // GREEK SMALL LETTER EPSILON WITH TONOS
	0x03AE: " eta ",       // GREEK SMALL LETTER ETA WITH TONOS
	0x03AF: " iota ",      // GREEK SMALL LETTER IOTA WITH TONOS
	0x03B0: " upsilon ",   // GREEK SMALL LETTER UPSILON WITH DIALYTIKA AND TONOS
	0x03B1: " alpha ",     // GREEK SMALL LETTER ALPHA
	0x03B2: " beta ",      // GREEK SMALL LETTER BETA
	0x03B3: " gamma ",     // GREEK SMALL LETTER GAMMA
	0x03B4: " delta ",     // GREEK SMALL LETTER DELTA
	0x03B5: " epsilon ",   // GREEK SMALL LETTER EPSILON
	0x03B6: " zeta ",      // GREEK SMALL LETTER ZETA
	0x03B7: " eta ",       // GREEK SMALL LETTER ETA
	0x03B8: " theta ",     // GREEK SMALL LETTER THETA
	0x03B9: " iota ",      // GREEK SMALL LETTER IOTA
	0x03BA: " kappa ",     // GREEK SMALL LETTER KAPPA
	0x03BB: " lambda ",    // GREEK SMALL LETTER LAMDA
	0x03BC: " mu ",        // GREEK SMALL LETTER MU
	0x03BD: " nu ",        // GREEK SMALL LETTER NU
	0x03BE: " xi ",        // GREEK SMALL LETTER XI
	0x03BF: " omicron ",   // GREEK SMALL LETTER OMICRON
	0x03C0: " pi ",        // GREEK SMALL LETTER PI
	0x03C1: " rho ",       // GREEK SMALL LETTER RHO
	0x03C2: " sigma ",     // GREEK SMALL LETTER FINAL SIGMA
	0x03C3: " sigma ",     // GREEK SMALL LETTER SIGMA
	0x03C4: " tau ",       // GREEK SMALL LETTER TAU
	0x03C5: " upsilon ",   // GREEK SMALL LETTER UPSILON
	0x03C6: " phi ",       // GREEK SMALL LETTER PHI
	0x03C7: " chi ",       // GREEK SMALL LETTER CHI
	0x03C8: " psi ",       // GREEK SMALL LETTER PSI
	0x03C9: " omega ",     // GREEK SMALL LETTER OMEGA
	0x03CA: " iota ",      // GREEK SMALL LETTER IOTA WITH DIALYTIKA
	0x03CB: " upsilon ",   // GREEK SMALL LETTER UPSILON WITH DIALYTIKA
	0x03CC: " omicron ",   // GREEK SMALL LETTER OMICRON WITH TONOS
	0x03CD: " upsilon ",   // GREEK SMALL LETTER UPSILON WITH TONOS
	0x03CE: " omega ",     // GREEK SMALL LETTER OMEGA WITH TONOS
	0x03CF: " Kai ",       //
	0x03D0: " beta ",      // GREEK BETA SYMBOL
	0x03D1: " theta ",     // GREEK THETA SYMBOL
	0x03D2: " Upsilon ",   // GREEK UPSILON WITH HOOK SYMBOL
	0x03D3: "",            // GREEK UPSILON WITH ACUTE AND HOOK SYMBOL
	0x03D4: " Upsilon ",   // GREEK UPSILON WITH DIAERESIS AND HOOK SYMBOL
	0x03D5: " Phi ",       // GREEK PHI SYMBOL
	0x03D6: " Pi ",        // GREEK PI SYMBOL
	0x03D7: " Kai ",       // GREEK KAI SYMBOL
	0x03DA: " Stigma ",    // GREEK LETTER STIGMA
	0x03DB: " stigma ",    // GREEK SMALL LETTER STIGMA
	0x03DC: " Digamma ",   // GREEK LETTER DIGAMMA
	0x03DD: " digamma ",   // GREEK SMALL LETTER DIGAMMA
	0x03DE: " Koppa ",     // GREEK LETTER KOPPA
	0x03DF: " koppa ",     // GREEK SMALL LETTER KOPPA
	0x03E0: " Sampi ",     // GREEK LETTER SAMPI
	0x03E1: " sampi ",     // GREEK SMALL LETTER SAMPI
	0x03E2: "Sh",          // COPTIC CAPITAL LETTER SHEI
	0x03E3: "sh",          // COPTIC SMALL LETTER SHEI
	0x03E4: "F",           // COPTIC CAPITAL LETTER FEI
	0x03E5: "f",           // COPTIC SMALL LETTER FEI
	0x03E6: "Kh",          // COPTIC CAPITAL LETTER KHEI
	0x03E7: "kh",          // COPTIC SMALL LETTER KHEI
	0x03E8: "H",           // COPTIC CAPITAL LETTER HORI
	0x03E9: "h",           // COPTIC SMALL LETTER HORI
	0x03EA: "G",           // COPTIC CAPITAL LETTER GANGIA
	0x03EB: "g",           // COPTIC SMALL LETTER GANGIA
	0x03EC: "CH",          // COPTIC CAPITAL LETTER SHIMA
	0x03ED: "ch",          // COPTIC SMALL LETTER SHIMA
	0x03EE: "Ti",          // COPTIC CAPITAL LETTER DEI
	0x03EF: "ti",          // COPTIC SMALL LETTER DEI
	0x03F0: "k",           // GREEK KAPPA SYMBOL
	0x03F1: "r",           // GREEK RHO SYMBOL
	0x03F2: "c",           // GREEK LUNATE SIGMA SYMBOL
	0x03F3: "j",           // GREEK LETTER YOT
	0x03F4: "Th",          //
	0x03F5: "e",           //
	0x03F6: "e",           //
	0x03F7: "",            //
	0x03F8: "",            //
	0x03F9: "S",           //
	0x03FA: "",            //
	0x03FB: "",            //
	0x03FC: "r",           //
	0x03FD: "S",           //
	0x03FE: "S",           //
	0x03FF: "S",           //
	0x0400: "Ie",          // CYRILLIC CAPITAL LETTER IE WITH GRAVE
	0x0401: "Io",          // CYRILLIC CAPITAL LETTER IO
	0x0402: "Dj",          // CYRILLIC CAPITAL LETTER DJE
	0x0403: "Gj",          // CYRILLIC CAPITAL LETTER GJE
	0x0404: "Ie",          // CYRILLIC CAPITAL LETTER UKRAINIAN IE
	0x0405: "Dz",          // CYRILLIC CAPITAL LETTER DZE
	0x0406: "I",           // CYRILLIC CAPITAL LETTER BYELORUSSIAN-UKRAINIAN I
	0x0407: "Yi",          // CYRILLIC CAPITAL LETTER YI
	0x0408: "J",           // CYRILLIC CAPITAL LETTER JE
	0x0409: "Lj",          // CYRILLIC CAPITAL LETTER LJE
	0x040A: "Nj",          // CYRILLIC CAPITAL LETTER NJE
	0x040B: "Tsh",         // CYRILLIC CAPITAL LETTER TSHE
	0x040C: "Kj",          // CYRILLIC CAPITAL LETTER KJE
	0x040D: "I",           // CYRILLIC CAPITAL LETTER I WITH GRAVE
	0x040E: "U",           // CYRILLIC CAPITAL LETTER SHORT U
	0x040F: "Dzh",         // CYRILLIC CAPITAL LETTER DZHE
	0x0410: "A",           // CYRILLIC CAPITAL LETTER A
	0x0411: "B",           // CYRILLIC CAPITAL LETTER BE
	0x0412: "V",           // CYRILLIC CAPITAL LETTER VE
	0x0413: "G",           // CYRILLIC CAPITAL LETTER GHE
	0x0414: "D",           // CYRILLIC CAPITAL LETTER DE
	0x0415: "E",           // CYRILLIC CAPITAL LETTER IE
	0x0416: "Zh",          // CYRILLIC CAPITAL LETTER ZHE
	0x0417: "Z",           // CYRILLIC CAPITAL LETTER ZE
	0x0418: "I",           // CYRILLIC CAPITAL LETTER I
	0x0419: "I",           // CYRILLIC CAPITAL LETTER SHORT I
	0x041A: "K",           // CYRILLIC CAPITAL LETTER KA
	0x041B: "L",           // CYRILLIC CAPITAL LETTER EL
	0x041C: "M",           // CYRILLIC CAPITAL LETTER EM
	0x041D: "N",           // CYRILLIC CAPITAL LETTER EN
	0x041E: "O",           // CYRILLIC CAPITAL LETTER O
	0x041F: "P",           // CYRILLIC CAPITAL LETTER PE
	0x0420: "R",           // CYRILLIC CAPITAL LETTER ER
	0x0421: "S",           // CYRILLIC CAPITAL LETTER ES
	0x0422: "T",           // CYRILLIC CAPITAL LETTER TE
	0x0423: "U",           // CYRILLIC CAPITAL LETTER U
	0x0424: "F",           // CYRILLIC CAPITAL LETTER EF
	0x0425: "Kh",          // CYRILLIC CAPITAL LETTER HA
	0x0426: "Ts",          // CYRILLIC CAPITAL LETTER TSE
	0x0427: "Ch",          // CYRILLIC CAPITAL LETTER CHE
	0x0428: "Sh",          // CYRILLIC CAPITAL LETTER SHA
	0x0429: "Shch",        // CYRILLIC CAPITAL LETTER SHCHA
	0x042A: "'",           // CYRILLIC CAPITAL LETTER HARD SIGN
	0x042B: "Y",           // CYRILLIC CAPITAL LETTER YERU
	0x042C: "'",           // CYRILLIC CAPITAL LETTER SOFT SIGN
	0x042D: "E",           // CYRILLIC CAPITAL LETTER E
	0x042E: "Iu",          // CYRILLIC CAPITAL LETTER YU
	0x042F: "Ia",          // CYRILLIC CAPITAL LETTER YA
	0x0430: "a",           // CYRILLIC SMALL LETTER A
	0x0431: "b",           // CYRILLIC SMALL LETTER BE
	0x0432: "v",           // CYRILLIC SMALL LETTER VE
	0x0433: "g",           // CYRILLIC SMALL LETTER GHE
	0x0434: "d",           // CYRILLIC SMALL LETTER DE
	0x0435: "e",           // CYRILLIC SMALL LETTER IE
	0x0436: "zh",          // CYRILLIC SMALL LETTER ZHE
	0x0437: "z",           // CYRILLIC SMALL LETTER ZE
	0x0438: "i",           // CYRILLIC SMALL LETTER I
	0x0439: "i",           // CYRILLIC SMALL LETTER SHORT I
	0x043A: "k",           // CYRILLIC SMALL LETTER KA
	0x043B: "l",           // CYRILLIC SMALL LETTER EL
	0x043C: "m",           // CYRILLIC SMALL LETTER EM
	0x043D: "n",           // CYRILLIC SMALL LETTER EN
	0x043E: "o",           // CYRILLIC SMALL LETTER O
	0x043F: "p",           // CYRILLIC SMALL LETTER PE
	0x0440: "r",           // CYRILLIC SMALL LETTER ER
	0x0441: "s",           // CYRILLIC SMALL LETTER ES
	0x0442: "t",           // CYRILLIC SMALL LETTER TE
	0x0443: "u",           // CYRILLIC SMALL LETTER U
	0x0444: "f",           // CYRILLIC SMALL LETTER EF
	0x0445: "kh",          // CYRILLIC SMALL LETTER HA
	0x0446: "ts",          // CYRILLIC SMALL LETTER TSE
	0x0447: "ch",          // CYRILLIC SMALL LETTER CHE
	0x0448: "sh",          // CYRILLIC SMALL LETTER SHA
	0x0449: "shch",        // CYRILLIC SMALL LETTER SHCHA
	0x044A: "'",           // CYRILLIC SMALL LETTER HARD SIGN
	0x044B: "y",           // CYRILLIC SMALL LETTER YERU
	0x044C: "'",           // CYRILLIC SMALL LETTER SOFT SIGN
	0x044D: "e",           // CYRILLIC SMALL LETTER E
	0x044E: "iu",          // CYRILLIC SMALL LETTER YU
	0x044F: "ia",          // CYRILLIC SMALL LETTER YA
	0x0450: "ie",          // CYRILLIC SMALL LETTER IE WITH GRAVE
	0x0451: "io",          // CYRILLIC SMALL LETTER IO
	0x0452: "dj",          // CYRILLIC SMALL LETTER DJE
	0x0453: "gj",          // CYRILLIC SMALL LETTER GJE
	0x0454: "ie",          // CYRILLIC SMALL LETTER UKRAINIAN IE
	0x0455: "dz",          // CYRILLIC SMALL LETTER DZE
	0x0456: "i",           // CYRILLIC SMALL LETTER BYELORUSSIAN-UKRAINIAN I
	0x0457: "yi",          // CYRILLIC SMALL LETTER YI
	0x0458: "j",           // CYRILLIC SMALL LETTER JE
	0x0459: "lj",          // CYRILLIC SMALL LETTER LJE
	0x045A: "nj",          // CYRILLIC SMALL LETTER NJE
	0x045B: "tsh",         // CYRILLIC SMALL LETTER TSHE
	0x045C: "kj",          // CYRILLIC SMALL LETTER KJE
	0x045D: "i",           // CYRILLIC SMALL LETTER I WITH GRAVE
	0x045E: "u",           // CYRILLIC SMALL LETTER SHORT U
	0x045F: "dzh",         // CYRILLIC SMALL LETTER DZHE
	0x0460: "O",           // CYRILLIC CAPITAL LETTER OMEGA
	0x0461: "o",           // CYRILLIC SMALL LETTER OMEGA
	0x0462: "E",           // CYRILLIC CAPITAL LETTER YAT
	0x0463: "e",           // CYRILLIC SMALL LETTER YAT
	0x0464: "Ie",          // CYRILLIC CAPITAL LETTER IOTIFIED E
	0x0465: "ie",          // CYRILLIC SMALL LETTER IOTIFIED E
	0x0466: "E",           // CYRILLIC CAPITAL LETTER LITTLE YUS
	0x0467: "e",           // CYRILLIC SMALL LETTER LITTLE YUS
	0x0468: "Ie",          // CYRILLIC CAPITAL LETTER IOTIFIED LITTLE YUS
	0x0469: "ie",          // CYRILLIC SMALL LETTER IOTIFIED LITTLE YUS
	0x046A: "O",           // CYRILLIC CAPITAL LETTER BIG YUS
	0x046B: "o",           // CYRILLIC SMALL LETTER BIG YUS
	0x046C: "Io",          // CYRILLIC CAPITAL LETTER IOTIFIED BIG YUS
	0x046D: "io",          // CYRILLIC SMALL LETTER IOTIFIED BIG YUS
	0x046E: "Ks",          // CYRILLIC CAPITAL LETTER KSI
	0x046F: "ks",          // CYRILLIC SMALL LETTER KSI
	0x0470: "Ps",          // CYRILLIC CAPITAL LETTER PSI
	0x0471: "ps",          // CYRILLIC SMALL LETTER PSI
	0x0472: "F",           // CYRILLIC CAPITAL LETTER FITA
	0x0473: "f",           // CYRILLIC SMALL LETTER FITA
	0x0474: "Y",           // CYRILLIC CAPITAL LETTER IZHITSA
	0x0475: "y",           // CYRILLIC SMALL LETTER IZHITSA
	0x0476: "Y",           // CYRILLIC CAPITAL LETTER IZHITSA WITH DOUBLE GRAVE ACCENT
	0x0477: "y",           // CYRILLIC SMALL LETTER IZHITSA WITH DOUBLE GRAVE ACCENT
	0x0478: "u",           // CYRILLIC CAPITAL LETTER UK
	0x0479: "u",           // CYRILLIC SMALL LETTER UK
	0x047A: "O",           // CYRILLIC CAPITAL LETTER ROUND OMEGA
	0x047B: "o",           // CYRILLIC SMALL LETTER ROUND OMEGA
	0x047C: "O",           // CYRILLIC CAPITAL LETTER OMEGA WITH TITLO
	0x047D: "o",           // CYRILLIC SMALL LETTER OMEGA WITH TITLO
	0x047E: "Ot",          // CYRILLIC CAPITAL LETTER OT
	0x047F: "ot",          // CYRILLIC SMALL LETTER OT
	0x0480: "Q",           // CYRILLIC CAPITAL LETTER KOPPA
	0x0481: "q",           // CYRILLIC SMALL LETTER KOPPA
	0x048C: "",            // CYRILLIC CAPITAL LETTER SEMISOFT SIGN
	0x048D: "",            // CYRILLIC SMALL LETTER SEMISOFT SIGN
	0x048E: "R'",          // CYRILLIC CAPITAL LETTER ER WITH TICK
	0x048F: "r'",          // CYRILLIC SMALL LETTER ER WITH TICK
	0x0490: "G'",          // CYRILLIC CAPITAL LETTER GHE WITH UPTURN
	0x0491: "g'",          // CYRILLIC SMALL LETTER GHE WITH UPTURN
	0x0492: "G'",          // CYRILLIC CAPITAL LETTER GHE WITH STROKE
	0x0493: "g'",          // CYRILLIC SMALL LETTER GHE WITH STROKE
	0x0494: "G'",          // CYRILLIC CAPITAL LETTER GHE WITH MIDDLE HOOK
	0x0495: "g'",          // CYRILLIC SMALL LETTER GHE WITH MIDDLE HOOK
	0x0496: "Zh'",         // CYRILLIC CAPITAL LETTER ZHE WITH DESCENDER
	0x0497: "zh'",         // CYRILLIC SMALL LETTER ZHE WITH DESCENDER
	0x0498: "Z'",          // CYRILLIC CAPITAL LETTER ZE WITH DESCENDER
	0x0499: "z'",          // CYRILLIC SMALL LETTER ZE WITH DESCENDER
	0x049A: "K'",          // CYRILLIC CAPITAL LETTER KA WITH DESCENDER
	0x049B: "k'",          // CYRILLIC SMALL LETTER KA WITH DESCENDER
	0x049C: "K'",          // CYRILLIC CAPITAL LETTER KA WITH VERTICAL STROKE
	0x049D: "k'",          // CYRILLIC SMALL LETTER KA WITH VERTICAL STROKE
	0x049E: "K'",          // CYRILLIC CAPITAL LETTER KA WITH STROKE
	0x049F: "k'",          // CYRILLIC SMALL LETTER KA WITH STROKE
	0x04A0: "K'",          // CYRILLIC CAPITAL LETTER BASHKIR KA
	0x04A1: "k'",          // CYRILLIC SMALL LETTER BASHKIR KA
	0x04A2: "N'",          // CYRILLIC CAPITAL LETTER EN WITH DESCENDER
	0x04A3: "n'",          // CYRILLIC SMALL LETTER EN WITH DESCENDER
	0x04A4: "Ng",          // CYRILLIC CAPITAL LIGATURE EN GHE
	0x04A5: "ng",          // CYRILLIC SMALL LIGATURE EN GHE
	0x04A6: "P'",          // CYRILLIC CAPITAL LETTER PE WITH MIDDLE HOOK
	0x04A7: "p'",          // CYRILLIC SMALL LETTER PE WITH MIDDLE HOOK
	0x04A8: "Kh",          // CYRILLIC CAPITAL LETTER ABKHASIAN HA
	0x04A9: "kh",          // CYRILLIC SMALL LETTER ABKHASIAN HA
	0x04AA: "S'",          // CYRILLIC CAPITAL LETTER ES WITH DESCENDER
	0x04AB: "s'",          // CYRILLIC SMALL LETTER ES WITH DESCENDER
	0x04AC: "T'",          // CYRILLIC CAPITAL LETTER TE WITH DESCENDER
	0x04AD: "t'",          // CYRILLIC SMALL LETTER TE WITH DESCENDER
	0x04AE: "U",           // CYRILLIC CAPITAL LETTER STRAIGHT U
	0x04AF: "u",           // CYRILLIC SMALL LETTER STRAIGHT U
	0x04B0: "U'",          // CYRILLIC CAPITAL LETTER STRAIGHT U WITH STROKE
	0x04B1: "u'",          // CYRILLIC SMALL LETTER STRAIGHT U WITH STROKE
	0x04B2: "Kh'",         // CYRILLIC CAPITAL LETTER HA WITH DESCENDER
	0x04B3: "kh'",         // CYRILLIC SMALL LETTER HA WITH DESCENDER
	0x04B4: "Tts",         // CYRILLIC CAPITAL LIGATURE TE TSE
	0x04B5: "tts",         // CYRILLIC SMALL LIGATURE TE TSE
	0x04B6: "Ch'",         // CYRILLIC CAPITAL LETTER CHE WITH DESCENDER
	0x04B7: "ch'",         // CYRILLIC SMALL LETTER CHE WITH DESCENDER
	0x04B8: "Ch'",         // CYRILLIC CAPITAL LETTER CHE WITH VERTICAL STROKE
	0x04B9: "ch'",         // CYRILLIC SMALL LETTER CHE WITH VERTICAL STROKE
	0x04BA: "H",           // CYRILLIC CAPITAL LETTER SHHA
	0x04BB: "h",           // CYRILLIC SMALL LETTER SHHA
	0x04BC: "Ch",          // CYRILLIC CAPITAL LETTER ABKHASIAN CHE
	0x04BD: "ch",          // CYRILLIC SMALL LETTER ABKHASIAN CHE
	0x04BE: "Ch'",         // CYRILLIC CAPITAL LETTER ABKHASIAN CHE WITH DESCENDER
	0x04BF: "ch'",         // CYRILLIC SMALL LETTER ABKHASIAN CHE WITH DESCENDER
	0x04C0: "`",           // CYRILLIC LETTER PALOCHKA
	0x04C1: "Zh",          // CYRILLIC CAPITAL LETTER ZHE WITH BREVE
	0x04C2: "zh",          // CYRILLIC SMALL LETTER ZHE WITH BREVE
	0x04C3: "K'",          // CYRILLIC CAPITAL LETTER KA WITH HOOK
	0x04C4: "k'",          // CYRILLIC SMALL LETTER KA WITH HOOK
	0x04C7: "N'",          // CYRILLIC CAPITAL LETTER EN WITH HOOK
	0x04C8: "n'",          // CYRILLIC SMALL LETTER EN WITH HOOK
	0x04CB: "Ch",          // CYRILLIC CAPITAL LETTER KHAKASSIAN CHE
	0x04CC: "ch",          // CYRILLIC SMALL LETTER KHAKASSIAN CHE
	0x04D0: "a",           // CYRILLIC CAPITAL LETTER A WITH BREVE
	0x04D1: "a",           // CYRILLIC SMALL LETTER A WITH BREVE
	0x04D2: "A",           // CYRILLIC CAPITAL LETTER A WITH DIAERESIS
	0x04D3: "a",           // CYRILLIC SMALL LETTER A WITH DIAERESIS
	0x04D4: "Ae",          // CYRILLIC CAPITAL LIGATURE A IE
	0x04D5: "ae",          // CYRILLIC SMALL LIGATURE A IE
	0x04D6: "Ie",          // CYRILLIC CAPITAL LETTER IE WITH BREVE
	0x04D7: "ie",          // CYRILLIC SMALL LETTER IE WITH BREVE
	0x04D8: "@",           // CYRILLIC CAPITAL LETTER SCHWA
	0x04D9: "@",           // CYRILLIC SMALL LETTER SCHWA
	0x04DA: "@",           // CYRILLIC CAPITAL LETTER SCHWA WITH DIAERESIS
	0x04DB: "@",           // CYRILLIC SMALL LETTER SCHWA WITH DIAERESIS
	0x04DC: "Zh",          // CYRILLIC CAPITAL LETTER ZHE WITH DIAERESIS
	0x04DD: "zh",          // CYRILLIC SMALL LETTER ZHE WITH DIAERESIS
	0x04DE: "Z",           // CYRILLIC CAPITAL LETTER ZE WITH DIAERESIS
	0x04DF: "z",           // CYRILLIC SMALL LETTER ZE WITH DIAERESIS
	0x04E0: "Dz",          // CYRILLIC CAPITAL LETTER ABKHASIAN DZE
	0x04E1: "dz",          // CYRILLIC SMALL LETTER ABKHASIAN DZE
	0x04E2: "I",           // CYRILLIC CAPITAL LETTER I WITH MACRON
	0x04E3: "i",           // CYRILLIC SMALL LETTER I WITH MACRON
	0x04E4: "I",           // CYRILLIC CAPITAL LETTER I WITH DIAERESIS
	0x04E5: "i",           // CYRILLIC SMALL LETTER I WITH DIAERESIS
	0x04E6: "O",           // CYRILLIC CAPITAL LETTER O WITH DIAERESIS
	0x04E7: "o",           // CYRILLIC SMALL LETTER O WITH DIAERESIS
	0x04E8: "O",           // CYRILLIC CAPITAL LETTER BARRED O
	0x04E9: "o",           // CYRILLIC SMALL LETTER BARRED O
	0x04EA: "O",           // CYRILLIC CAPITAL LETTER BARRED O WITH DIAERESIS
	0x04EB: "o",           // CYRILLIC SMALL LETTER BARRED O WITH DIAERESIS
	0x04EC: "E",           // CYRILLIC CAPITAL LETTER E WITH DIAERESIS
	0x04ED: "e",           // CYRILLIC SMALL LETTER E WITH DIAERESIS
	0x04EE: "U",           // CYRILLIC CAPITAL LETTER U WITH MACRON
	0x04EF: "u",           // CYRILLIC SMALL LETTER U WITH MACRON
	0x04F0: "U",           // CYRILLIC CAPITAL LETTER U WITH DIAERESIS
	0x04F1: "u",           // CYRILLIC SMALL LETTER U WITH DIAERESIS
	0x04F2: "U",           // CYRILLIC CAPITAL LETTER U WITH DOUBLE ACUTE
	0x04F3: "u",           // CYRILLIC SMALL LETTER U WITH DOUBLE ACUTE
	0x04F4: "Ch",          // CYRILLIC CAPITAL LETTER CHE WITH DIAERESIS
	0x04F5: "ch",          // CYRILLIC SMALL LETTER CHE WITH DIAERESIS
	0x04F8: "Y",           // CYRILLIC CAPITAL LETTER YERU WITH DIAERESIS
	0x04F9: "y",           // CYRILLIC SMALL LETTER YERU WITH DIAERESIS
	0x0531: "A",           // ARMENIAN CAPITAL LETTER AYB
	0x0532: "B",           // ARMENIAN CAPITAL LETTER BEN
	0x0533: "G",           // ARMENIAN CAPITAL LETTER GIM
	0x0534: "D",           // ARMENIAN CAPITAL LETTER DA
	0x0535: "E",           // ARMENIAN CAPITAL LETTER ECH
	0x0536: "Z",           // ARMENIAN CAPITAL LETTER ZA
	0x0537: "E",           // ARMENIAN CAPITAL LETTER EH
	0x0538: "E",           // ARMENIAN CAPITAL LETTER ET
	0x0539: "T`",          // ARMENIAN CAPITAL LETTER TO
	0x053A: "Zh",          // ARMENIAN CAPITAL LETTER ZHE
	0x053B: "I",           // ARMENIAN CAPITAL LETTER INI
	0x053C: "L",           // ARMENIAN CAPITAL LETTER LIWN
	0x053D: "Kh",          // ARMENIAN CAPITAL LETTER XEH
	0x053E: "Ts",          // ARMENIAN CAPITAL LETTER CA
	0x053F: "K",           // ARMENIAN CAPITAL LETTER KEN
	0x0540: "H",           // ARMENIAN CAPITAL LETTER HO
	0x0541: "Dz",          // ARMENIAN CAPITAL LETTER JA
	0x0542: "Gh",          // ARMENIAN CAPITAL LETTER GHAD
	0x0543: "Ch",          // ARMENIAN CAPITAL LETTER CHEH
	0x0544: "M",           // ARMENIAN CAPITAL LETTER MEN
	0x0545: "Y",           // ARMENIAN CAPITAL LETTER YI
	0x0546: "N",           // ARMENIAN CAPITAL LETTER NOW
	0x0547: "Sh",          // ARMENIAN CAPITAL LETTER SHA
	0x0548: "O",           // ARMENIAN CAPITAL LETTER VO
	0x0549: "Ch`",         // ARMENIAN CAPITAL LETTER CHA
	0x054A: "P",           // ARMENIAN CAPITAL LETTER PEH
	0x054B: "J",           // ARMENIAN CAPITAL LETTER JHEH
	0x054C: "Rh",          // ARMENIAN CAPITAL LETTER RA
	0x054D: "S",           // ARMENIAN CAPITAL LETTER SEH
	0x054E: "V",           // ARMENIAN CAPITAL LETTER VEW
	0x054F: "T",           // ARMENIAN CAPITAL LETTER TIWN
	0x0550: "R",           // ARMENIAN CAPITAL LETTER REH
	0x0551: "Ts`",         // ARMENIAN CAPITAL LETTER CO
	0x0552: "W",           // ARMENIAN CAPITAL LETTER YIWN
	0x0553: "P`",          // ARMENIAN CAPITAL LETTER PIWR
	0x0554: "K`",          // ARMENIAN CAPITAL LETTER KEH
	0x0555: "O",           // ARMENIAN CAPITAL LETTER OH
	0x0556: "F",           // ARMENIAN CAPITAL LETTER FEH
	0x0559: "<",           // ARMENIAN MODIFIER LETTER LEFT HALF RING
	0x055A: "'",           // ARMENIAN APOSTROPHE
	0x055B: "/",           // ARMENIAN EMPHASIS MARK
	0x055C: "!",           // ARMENIAN EXCLAMATION MARK
	0x055D: ",",           // ARMENIAN COMMA
	0x055E: "?",           // ARMENIAN QUESTION MARK
	0x055F: ".",           // ARMENIAN ABBREVIATION MARK
	0x0561: "a",           // ARMENIAN SMALL LETTER AYB
	0x0562: "b",           // ARMENIAN SMALL LETTER BEN
	0x0563: "g",           // ARMENIAN SMALL LETTER GIM
	0x0564: "d",           // ARMENIAN SMALL LETTER DA
	0x0565: "e",           // ARMENIAN SMALL LETTER ECH
	0x0566: "z",           // ARMENIAN SMALL LETTER ZA
	0x0567: "e",           // ARMENIAN SMALL LETTER EH
	0x0568: "e",           // ARMENIAN SMALL LETTER ET
	0x0569: "t`",          // ARMENIAN SMALL LETTER TO
	0x056A: "zh",          // ARMENIAN SMALL LETTER ZHE
	0x056B: "i",           // ARMENIAN SMALL LETTER INI
	0x056C: "l",           // ARMENIAN SMALL LETTER LIWN
	0x056D: "kh",          // ARMENIAN SMALL LETTER XEH
	0x056E: "ts",          // ARMENIAN SMALL LETTER CA
	0x056F: "k",           // ARMENIAN SMALL LETTER KEN
	0x0570: "h",           // ARMENIAN SMALL LETTER HO
	0x0571: "dz",          // ARMENIAN SMALL LETTER JA
	0x0572: "gh",          // ARMENIAN SMALL LETTER GHAD
	0x0573: "ch",          // ARMENIAN SMALL LETTER CHEH
	0x0574: "m",           // ARMENIAN SMALL LETTER MEN
	0x0575: "y",           // ARMENIAN SMALL LETTER YI
	0x0576: "n",           // ARMENIAN SMALL LETTER NOW
	0x0577: "sh",          // ARMENIAN SMALL LETTER SHA
	0x0578: "o",           // ARMENIAN SMALL LETTER VO
	0x0579: "ch`",         // ARMENIAN SMALL LETTER CHA
	0x057A: "p",           // ARMENIAN SMALL LETTER PEH
	0x057B: "j",           // ARMENIAN SMALL LETTER JHEH
	0x057C: "rh",          // ARMENIAN SMALL LETTER RA
	0x057D: "s",           // ARMENIAN SMALL LETTER SEH
	0x057E: "v",           // ARMENIAN SMALL LETTER VEW
	0x057F: "t",           // ARMENIAN SMALL LETTER TIWN
	0x0580: "r",           // ARMENIAN SMALL LETTER REH
	0x0581: "ts`",         // ARMENIAN SMALL LETTER CO
	0x0582: "w",           // ARMENIAN SMALL LETTER YIWN
	0x0583: "p`",          // ARMENIAN SMALL LETTER PIWR
	0x0584: "k`",          // ARMENIAN SMALL LETTER KEH
	0x0585: "o",           // ARMENIAN SMALL LETTER OH
	0x0586: "f",           // ARMENIAN SMALL LETTER FEH
	0x0587: "ew",          // ARMENIAN SMALL LIGATURE ECH YIWN
	0x0589: ":",           // ARMENIAN FULL STOP
	0x058A: "-",           // ARMENIAN HYPHEN
	0x05B0: "@",           // HEBREW POINT SHEVA
	0x05B1: "e",           // HEBREW POINT HATAF SEGOL
	0x05B2: "a",           // HEBREW POINT HATAF PATAH
	0x05B3: "o",           // HEBREW POINT HATAF QAMATS
	0x05B4: "i",           // HEBREW POINT HIRIQ
	0x05B5: "e",           // HEBREW POINT TSERE
	0x05B6: "e",           // HEBREW POINT SEGOL
	0x05B7: "a",           // HEBREW POINT PATAH
	0x05B8: "a",           // HEBREW POINT QAMATS
	0x05B9: "o",           // HEBREW POINT HOLAM
	0x05BB: "u",           // HEBREW POINT QUBUTS
	0x05BC: "'",           // HEBREW POINT DAGESH OR MAPIQ
	0x05C0: "|",           // HEBREW PUNCTUATION PASEQ
	0x05C3: ":",           // HEBREW PUNCTUATION SOF PASUQ
	0x05D0: "a",           // HEBREW LETTER ALEF
	0x05D1: "b",           // HEBREW LETTER BET
	0x05D2: "g",           // HEBREW LETTER GIMEL
	0x05D3: "d",           // HEBREW LETTER DALET
	0x05D4: "h",           // HEBREW LETTER HE
	0x05D5: "v",           // HEBREW LETTER VAV
	0x05D6: "z",           // HEBREW LETTER ZAYIN
	0x05D7: "kh",          // HEBREW LETTER HET
	0x05D8: "t",           // HEBREW LETTER TET
	0x05D9: "y",           // HEBREW LETTER YOD
	0x05DA: "k",           // HEBREW LETTER FINAL KAF
	0x05DB: "k",           // HEBREW LETTER KAF
	0x05DC: "l",           // HEBREW LETTER LAMED
	0x05DD: "m",           // HEBREW LETTER FINAL MEM
	0x05DE: "m",           // HEBREW LETTER MEM
	0x05DF: "n",           // HEBREW LETTER FINAL NUN
	0x05E0: "n",           // HEBREW LETTER NUN
	0x05E1: "s",           // HEBREW LETTER SAMEKH
	0x05E2: "`",           // HEBREW LETTER AYIN
	0x05E3: "p",           // HEBREW LETTER FINAL PE
	0x05E4: "p",           // HEBREW LETTER PE
	0x05E5: "ts",          // HEBREW LETTER FINAL TSADI
	0x05E6: "ts",          // HEBREW LETTER TSADI
	0x05E7: "q",           // HEBREW LETTER QOF
	0x05E8: "r",           // HEBREW LETTER RESH
	0x05E9: "sh",          // HEBREW LETTER SHIN
	0x05EA: "t",           // HEBREW LETTER TAV
	0x05F0: "V",           // HEBREW LIGATURE YIDDISH DOUBLE VAV
	0x05F1: "oy",          // HEBREW LIGATURE YIDDISH VAV YOD
	0x05F2: "i",           // HEBREW LIGATURE YIDDISH DOUBLE YOD
	0x05F3: "'",           // HEBREW PUNCTUATION GERESH
	0x05F4: "",            // HEBREW PUNCTUATION GERSHAYIM
	0x060C: ",",           // ARABIC COMMA
	0x061B: ";",           // ARABIC SEMICOLON
	0x061F: "?",           // ARABIC QUESTION MARK
	0x0622: "a",           // ARABIC LETTER ALEF WITH MADDA ABOVE
	0x0623: "'",           // ARABIC LETTER ALEF WITH HAMZA ABOVE
	0x0624: "w'",          // ARABIC LETTER WAW WITH HAMZA ABOVE
	0x0626: "y'",          // ARABIC LETTER YEH WITH HAMZA ABOVE
	0x0628: "b",           // ARABIC LETTER BEH
	0x0629: "@",           // ARABIC LETTER TEH MARBUTA
	0x062A: "t",           // ARABIC LETTER TEH
	0x062B: "th",          // ARABIC LETTER THEH
	0x062C: "j",           // ARABIC LETTER JEEM
	0x062D: "H",           // ARABIC LETTER HAH
	0x062E: "kh",          // ARABIC LETTER KHAH
	0x062F: "d",           // ARABIC LETTER DAL
	0x0630: "dh",          // ARABIC LETTER THAL
	0x0631: "r",           // ARABIC LETTER REH
	0x0632: "z",           // ARABIC LETTER ZAIN
	0x0633: "s",           // ARABIC LETTER SEEN
	0x0634: "sh",          // ARABIC LETTER SHEEN
	0x0635: "S",           // ARABIC LETTER SAD
	0x0636: "D",           // ARABIC LETTER DAD
	0x0637: "T",           // ARABIC LETTER TAH
	0x0638: "Z",           // ARABIC LETTER ZAH
	0x0639: "`",           // ARABIC LETTER AIN
	0x063A: "G",           // ARABIC LETTER GHAIN
	0x0641: "f",           // ARABIC LETTER FEH
	0x0642: "q",           // ARABIC LETTER QAF
	0x0643: "k",           // ARABIC LETTER KAF
	0x0644: "l",           // ARABIC LETTER LAM
	0x0645: "m",           // ARABIC LETTER MEEM
	0x0646: "n",           // ARABIC LETTER NOON
	0x0647: "h",           // ARABIC LETTER HEH
	0x0648: "w",           // ARABIC LETTER WAW
	0x0649: "~",           // ARABIC LETTER ALEF MAKSURA
	0x064A: "y",           // ARABIC LETTER YEH
	0x064B: "an",          // ARABIC FATHATAN
	0x064C: "un",          // ARABIC DAMMATAN
	0x064D: "in",          // ARABIC KASRATAN
	0x064E: "a",           // ARABIC FATHA
	0x064F: "u",           // ARABIC DAMMA
	0x0650: "i",           // ARABIC KASRA
	0x0651: "W",           // ARABIC SHADDA
	0x0654: "'",           // ARABIC HAMZA ABOVE
	0x0655: "'",           // ARABIC HAMZA BELOW
	0x0660: "0",           // ARABIC-INDIC DIGIT ZERO
	0x0661: "1",           // ARABIC-INDIC DIGIT ONE
	0x0662: "2",           // ARABIC-INDIC DIGIT TWO
	0x0663: "3",           // ARABIC-INDIC DIGIT THREE
	0x0664: "4",           // ARABIC-INDIC DIGIT FOUR
	0x0665: "5",           // ARABIC-INDIC DIGIT FIVE
	0x0666: "6",           // ARABIC-INDIC DIGIT SIX
	0x0667: "7",           // ARABIC-INDIC DIGIT SEVEN
	0x0668: "8",           // ARABIC-INDIC DIGIT EIGHT
	0x0669: "9",           // ARABIC-INDIC DIGIT NINE
	0x066A: "%",           // ARABIC PERCENT SIGN
	0x066B: ".",           // ARABIC DECIMAL SEPARATOR
	0x066C: ",",           // ARABIC THOUSANDS SEPARATOR
	0x066D: "*",           // ARABIC FIVE POINTED STAR
	0x0671: "'",           // ARABIC LETTER ALEF WASLA
	0x0672: "'",           // ARABIC LETTER ALEF WITH WAVY HAMZA ABOVE
	0x0673: "'",           // ARABIC LETTER ALEF WITH WAVY HAMZA BELOW
	0x0675: "'",           // ARABIC LETTER HIGH HAMZA ALEF
	0x0676: "'w",          // ARABIC LETTER HIGH HAMZA WAW
	0x0677: "'u",          // ARABIC LETTER U WITH HAMZA ABOVE
	0x0678: "'y",          // ARABIC LETTER HIGH HAMZA YEH
	0x0679: "tt",          // ARABIC LETTER TTEH
	0x067A: "tth",         // ARABIC LETTER TTEHEH
	0x067B: "b",           // ARABIC LETTER BEEH
	0x067C: "t",           // ARABIC LETTER TEH WITH RING
	0x067D: "T",           // ARABIC LETTER TEH WITH THREE DOTS ABOVE DOWNWARDS
	0x067E: "p",           // ARABIC LETTER PEH
	0x067F: "th",          // ARABIC LETTER TEHEH
	0x0680: "bh",          // ARABIC LETTER BEHEH
	0x0681: "'h",          // ARABIC LETTER HAH WITH HAMZA ABOVE
	0x0682: "H",           // ARABIC LETTER HAH WITH TWO DOTS VERTICAL ABOVE
	0x0683: "ny",          // ARABIC LETTER NYEH
	0x0684: "dy",          // ARABIC LETTER DYEH
	0x0685: "H",           // ARABIC LETTER HAH WITH THREE DOTS ABOVE
	0x0686: "ch",          // ARABIC LETTER TCHEH
	0x0687: "cch",         // ARABIC LETTER TCHEHEH
	0x0688: "dd",          // ARABIC LETTER DDAL
	0x0689: "D",           // ARABIC LETTER DAL WITH RING
	0x068A: "D",           // ARABIC LETTER DAL WITH DOT BELOW
	0x068B: "Dt",          // ARABIC LETTER DAL WITH DOT BELOW AND SMALL TAH
	0x068C: "dh",          // ARABIC LETTER DAHAL
	0x068D: "ddh",         // ARABIC LETTER DDAHAL
	0x068E: "d",           // ARABIC LETTER DUL
	0x068F: "D",           // ARABIC LETTER DAL WITH THREE DOTS ABOVE DOWNWARDS
	0x0690: "D",           // ARABIC LETTER DAL WITH FOUR DOTS ABOVE
	0x0691: "rr",          // ARABIC LETTER RREH
	0x0692: "R",           // ARABIC LETTER REH WITH SMALL V
	0x0693: "R",           // ARABIC LETTER REH WITH RING
	0x0694: "R",           // ARABIC LETTER REH WITH DOT BELOW
	0x0695: "R",           // ARABIC LETTER REH WITH SMALL V BELOW
	0x0696: "R",           // ARABIC LETTER REH WITH DOT BELOW AND DOT ABOVE
	0x0697: "R",           // ARABIC LETTER REH WITH TWO DOTS ABOVE
	0x0698: "j",           // ARABIC LETTER JEH
	0x0699: "R",           // ARABIC LETTER REH WITH FOUR DOTS ABOVE
	0x069A: "S",           // ARABIC LETTER SEEN WITH DOT BELOW AND DOT ABOVE
	0x069B: "S",           // ARABIC LETTER SEEN WITH THREE DOTS BELOW
	0x069C: "S",           // ARABIC LETTER SEEN WITH THREE DOTS BELOW AND THREE DOTS ABOVE
	0x069D: "S",           // ARABIC LETTER SAD WITH TWO DOTS BELOW
	0x069E: "S",           // ARABIC LETTER SAD WITH THREE DOTS ABOVE
	0x069F: "T",           // ARABIC LETTER TAH WITH THREE DOTS ABOVE
	0x06A0: "GH",          // ARABIC LETTER AIN WITH THREE DOTS ABOVE
	0x06A1: "F",           // ARABIC LETTER DOTLESS FEH
	0x06A2: "F",           // ARABIC LETTER FEH WITH DOT MOVED BELOW
	0x06A3: "F",           // ARABIC LETTER FEH WITH DOT BELOW
	0x06A4: "v",           // ARABIC LETTER VEH
	0x06A5: "f",           // ARABIC LETTER FEH WITH THREE DOTS BELOW
	0x06A6: "ph",          // ARABIC LETTER PEHEH
	0x06A7: "Q",           // ARABIC LETTER QAF WITH DOT ABOVE
	0x06A8: "Q",           // ARABIC LETTER QAF WITH THREE DOTS ABOVE
	0x06A9: "kh",          // ARABIC LETTER KEHEH
	0x06AA: "k",           // ARABIC LETTER SWASH KAF
	0x06AB: "K",           // ARABIC LETTER KAF WITH RING
	0x06AC: "K",           // ARABIC LETTER KAF WITH DOT ABOVE
	0x06AD: "ng",          // ARABIC LETTER NG
	0x06AE: "K",           // ARABIC LETTER KAF WITH THREE DOTS BELOW
	0x06AF: "g",           // ARABIC LETTER GAF
	0x06B0: "G",           // ARABIC LETTER GAF WITH RING
	0x06B1: "N",           // ARABIC LETTER NGOEH
	0x06B2: "G",           // ARABIC LETTER GAF WITH TWO DOTS BELOW
	0x06B3: "G",           // ARABIC LETTER GUEH
	0x06B4: "G",           // ARABIC LETTER GAF WITH THREE DOTS ABOVE
	0x06B5: "L",           // ARABIC LETTER LAM WITH SMALL V
	0x06B6: "L",           // ARABIC LETTER LAM WITH DOT ABOVE
	0x06B7: "L",           // ARABIC LETTER LAM WITH THREE DOTS ABOVE
	0x06B8: "L",           // ARABIC LETTER LAM WITH THREE DOTS BELOW
	0x06B9: "N",           // ARABIC LETTER NOON WITH DOT BELOW
	0x06BA: "N",           // ARABIC LETTER NOON GHUNNA
	0x06BB: "N",           // ARABIC LETTER RNOON
	0x06BC: "N",           // ARABIC LETTER NOON WITH RING
	0x06BD: "N",           // ARABIC LETTER NOON WITH THREE DOTS ABOVE
	0x06BE: "h",           // ARABIC LETTER HEH DOACHASHMEE
	0x06BF: "Ch",          // ARABIC LETTER TCHEH WITH DOT ABOVE
	0x06C0: "hy",          // ARABIC LETTER HEH WITH YEH ABOVE
	0x06C1: "h",           // ARABIC LETTER HEH GOAL
	0x06C2: "H",           // ARABIC LETTER HEH GOAL WITH HAMZA ABOVE
	0x06C3: "@",           // ARABIC LETTER TEH MARBUTA GOAL
	0x06C4: "W",           // ARABIC LETTER WAW WITH RING
	0x06C5: "oe",          // ARABIC LETTER KIRGHIZ OE
	0x06C6: "oe",          // ARABIC LETTER OE
	0x06C7: "u",           // ARABIC LETTER U
	0x06C8: "yu",          // ARABIC LETTER YU
	0x06C9: "yu",          // ARABIC LETTER KIRGHIZ YU
	0x06CA: "W",           // ARABIC LETTER WAW WITH TWO DOTS ABOVE
	0x06CB: "v",           // ARABIC LETTER VE
	0x06CC: "y",           // ARABIC LETTER FARSI YEH
	0x06CD: "Y",           // ARABIC LETTER YEH WITH TAIL
	0x06CE: "Y",           // ARABIC LETTER YEH WITH SMALL V
	0x06CF: "W",           // ARABIC LETTER WAW WITH DOT ABOVE
	0x06D2: "y",           // ARABIC LETTER YEH BARREE
	0x06D3: "y'",          // ARABIC LETTER YEH BARREE WITH HAMZA ABOVE
	0x06D4: ".",           // ARABIC FULL STOP
	0x06D5: "ae",          // ARABIC LETTER AE
	0x06DD: "@",           // ARABIC END OF AYAH
	0x06DE: "#",           // ARABIC START OF RUB EL HIZB
	0x06E9: "^",           // ARABIC PLACE OF SAJDAH
	0x06F0: "0",           // EXTENDED ARABIC-INDIC DIGIT ZERO
	0x06F1: "1",           // EXTENDED ARABIC-INDIC DIGIT ONE
	0x06F2: "2",           // EXTENDED ARABIC-INDIC DIGIT TWO
	0x06F3: "3",           // EXTENDED ARABIC-INDIC DIGIT THREE
	0x06F4: "4",           // EXTENDED ARABIC-INDIC DIGIT FOUR
	0x06F5: "5",           // EXTENDED ARABIC-INDIC DIGIT FIVE
	0x06F6: "6",           // EXTENDED ARABIC-INDIC DIGIT SIX
	0x06F7: "7",           // EXTENDED ARABIC-INDIC DIGIT SEVEN
	0x06F8: "8",           // EXTENDED ARABIC-INDIC DIGIT EIGHT
	0x06F9: "9",           // EXTENDED ARABIC-INDIC DIGIT NINE
	0x06FA: "Sh",          // ARABIC LETTER SHEEN WITH DOT BELOW
	0x06FB: "D",           // ARABIC LETTER DAD WITH DOT BELOW
	0x06FC: "Gh",          // ARABIC LETTER GHAIN WITH DOT BELOW
	0x06FD: "&",           // ARABIC SIGN SINDHI AMPERSAND
	0x06FE: "+m",          // ARABIC SIGN SINDHI POSTPOSITION MEN
	0x06FF: "",            //
	0x0700: "//",          // SYRIAC END OF PARAGRAPH
	0x0701: "/",           // SYRIAC SUPRALINEAR FULL STOP
	0x0702: ",",           // SYRIAC SUBLINEAR FULL STOP
	0x0703: "!",           // SYRIAC SUPRALINEAR COLON
	0x0704: "!",           // SYRIAC SUBLINEAR COLON
	0x0705: "-",           // SYRIAC HORIZONTAL COLON
	0x0706: ",",           // SYRIAC COLON SKEWED LEFT
	0x0707: ",",           // SYRIAC COLON SKEWED RIGHT
	0x0708: ";",           // SYRIAC SUPRALINEAR COLON SKEWED LEFT
	0x0709: "?",           // SYRIAC SUBLINEAR COLON SKEWED RIGHT
	0x070A: "~",           // SYRIAC CONTRACTION
	0x070B: "{",           // SYRIAC HARKLEAN OBELUS
	0x070C: "}",           // SYRIAC HARKLEAN METOBELUS
	0x070D: "*",           // SYRIAC HARKLEAN ASTERISCUS
	0x0712: "b",           // SYRIAC LETTER BETH
	0x0713: "g",           // SYRIAC LETTER GAMAL
	0x0714: "g",           // SYRIAC LETTER GAMAL GARSHUNI
	0x0715: "d",           // SYRIAC LETTER DALATH
	0x0716: "d",           // SYRIAC LETTER DOTLESS DALATH RISH
	0x0717: "h",           // SYRIAC LETTER HE
	0x0718: "w",           // SYRIAC LETTER WAW
	0x0719: "z",           // SYRIAC LETTER ZAIN
	0x071A: "H",           // SYRIAC LETTER HETH
	0x071B: "t",           // SYRIAC LETTER TETH
	0x071C: "t",           // SYRIAC LETTER TETH GARSHUNI
	0x071D: "y",           // SYRIAC LETTER YUDH
	0x071E: "yh",          // SYRIAC LETTER YUDH HE
	0x071F: "k",           // SYRIAC LETTER KAPH
	0x0720: "l",           // SYRIAC LETTER LAMADH
	0x0721: "m",           // SYRIAC LETTER MIM
	0x0722: "n",           // SYRIAC LETTER NUN
	0x0723: "s",           // SYRIAC LETTER SEMKATH
	0x0724: "s",           // SYRIAC LETTER FINAL SEMKATH
	0x0725: "`",           // SYRIAC LETTER E
	0x0726: "p",           // SYRIAC LETTER PE
	0x0727: "p",           // SYRIAC LETTER REVERSED PE
	0x0728: "S",           // SYRIAC LETTER SADHE
	0x0729: "q",           // SYRIAC LETTER QAPH
	0x072A: "r",           // SYRIAC LETTER RISH
	0x072B: "sh",          // SYRIAC LETTER SHIN
	0x072C: "t",           // SYRIAC LETTER TAW
	0x0730: "a",           // SYRIAC PTHAHA ABOVE
	0x0731: "a",           // SYRIAC PTHAHA BELOW
	0x0732: "a",           // SYRIAC PTHAHA DOTTED
	0x0733: "A",           // SYRIAC ZQAPHA ABOVE
	0x0734: "A",           // SYRIAC ZQAPHA BELOW
	0x0735: "A",           // SYRIAC ZQAPHA DOTTED
	0x0736: "e",           // SYRIAC RBASA ABOVE
	0x0737: "e",           // SYRIAC RBASA BELOW
	0x0738: "e",           // SYRIAC DOTTED ZLAMA HORIZONTAL
	0x0739: "E",           // SYRIAC DOTTED ZLAMA ANGULAR
	0x073A: "i",           // SYRIAC HBASA ABOVE
	0x073B: "i",           // SYRIAC HBASA BELOW
	0x073C: "u",           // SYRIAC HBASA-ESASA DOTTED
	0x073D: "u",           // SYRIAC ESASA ABOVE
	0x073E: "u",           // SYRIAC ESASA BELOW
	0x073F: "o",           // SYRIAC RWAHA
	0x0741: "`",           // SYRIAC QUSHSHAYA
	0x0742: "'",           // SYRIAC RUKKAKHA
	0x0745: "X",           // SYRIAC THREE DOTS ABOVE
	0x0746: "Q",           // SYRIAC THREE DOTS BELOW
	0x0747: "@",           // SYRIAC OBLIQUE LINE ABOVE
	0x0748: "@",           // SYRIAC OBLIQUE LINE BELOW
	0x0749: "|",           // SYRIAC MUSIC
	0x074A: "+",           // SYRIAC BARREKH
	0x0780: "h",           // THAANA LETTER HAA
	0x0781: "sh",          // THAANA LETTER SHAVIYANI
	0x0782: "n",           // THAANA LETTER NOONU
	0x0783: "r",           // THAANA LETTER RAA
	0x0784: "b",           // THAANA LETTER BAA
	0x0785: "L",           // THAANA LETTER LHAVIYANI
	0x0786: "k",           // THAANA LETTER KAAFU
	0x0787: "'",           // THAANA LETTER ALIFU
	0x0788: "v",           // THAANA LETTER VAAVU
	0x0789: "m",           // THAANA LETTER MEEMU
	0x078A: "f",           // THAANA LETTER FAAFU
	0x078B: "dh",          // THAANA LETTER DHAALU
	0x078C: "th",          // THAANA LETTER THAA
	0x078D: "l",           // THAANA LETTER LAAMU
	0x078E: "g",           // THAANA LETTER GAAFU
	0x078F: "ny",          // THAANA LETTER GNAVIYANI
	0x0790: "s",           // THAANA LETTER SEENU
	0x0791: "d",           // THAANA LETTER DAVIYANI
	0x0792: "z",           // THAANA LETTER ZAVIYANI
	0x0793: "t",           // THAANA LETTER TAVIYANI
	0x0794: "y",           // THAANA LETTER YAA
	0x0795: "p",           // THAANA LETTER PAVIYANI
	0x0796: "j",           // THAANA LETTER JAVIYANI
	0x0797: "ch",          // THAANA LETTER CHAVIYANI
	0x0798: "tt",          // THAANA LETTER TTAA
	0x0799: "hh",          // THAANA LETTER HHAA
	0x079A: "kh",          // THAANA LETTER KHAA
	0x079B: "th",          // THAANA LETTER THAALU
	0x079C: "z",           // THAANA LETTER ZAA
	0x079D: "sh",          // THAANA LETTER SHEENU
	0x079E: "s",           // THAANA LETTER SAADHU
	0x079F: "d",           // THAANA LETTER DAADHU
	0x07A0: "t",           // THAANA LETTER TO
	0x07A1: "z",           // THAANA LETTER ZO
	0x07A2: "`",           // THAANA LETTER AINU
	0x07A3: "gh",          // THAANA LETTER GHAINU
	0x07A4: "q",           // THAANA LETTER QAAFU
	0x07A5: "w",           // THAANA LETTER WAAVU
	0x07A6: "a",           // THAANA ABAFILI
	0x07A7: "aa",          // THAANA AABAAFILI
	0x07A8: "i",           // THAANA IBIFILI
	0x07A9: "ee",          // THAANA EEBEEFILI
	0x07AA: "u",           // THAANA UBUFILI
	0x07AB: "oo",          // THAANA OOBOOFILI
	0x07AC: "e",           // THAANA EBEFILI
	0x07AD: "ey",          // THAANA EYBEYFILI
	0x07AE: "o",           // THAANA OBOFILI
	0x07AF: "oa",          // THAANA OABOAFILI
	0x0901: "N",           // DEVANAGARI SIGN CANDRABINDU
	0x0902: "N",           // DEVANAGARI SIGN ANUSVARA
	0x0903: "H",           // DEVANAGARI SIGN VISARGA
	0x0905: "a",           // DEVANAGARI LETTER A
	0x0906: "aa",          // DEVANAGARI LETTER AA
	0x0907: "i",           // DEVANAGARI LETTER I
	0x0908: "ii",          // DEVANAGARI LETTER II
	0x0909: "u",           // DEVANAGARI LETTER U
	0x090A: "uu",          // DEVANAGARI LETTER UU
	0x090B: "R",           // DEVANAGARI LETTER VOCALIC R
	0x090C: "L",           // DEVANAGARI LETTER VOCALIC L
	0x090D: "eN",          // DEVANAGARI LETTER CANDRA E
	0x090E: "e",           // DEVANAGARI LETTER SHORT E
	0x090F: "e",           // DEVANAGARI LETTER E
	0x0910: "ai",          // DEVANAGARI LETTER AI
	0x0911: "oN",          // DEVANAGARI LETTER CANDRA O
	0x0912: "o",           // DEVANAGARI LETTER SHORT O
	0x0913: "o",           // DEVANAGARI LETTER O
	0x0914: "au",          // DEVANAGARI LETTER AU
	0x0915: "k",           // DEVANAGARI LETTER KA
	0x0916: "kh",          // DEVANAGARI LETTER KHA
	0x0917: "g",           // DEVANAGARI LETTER GA
	0x0918: "gh",          // DEVANAGARI LETTER GHA
	0x0919: "ng",          // DEVANAGARI LETTER NGA
	0x091A: "c",           // DEVANAGARI LETTER CA
	0x091B: "ch",          // DEVANAGARI LETTER CHA
	0x091C: "j",           // DEVANAGARI LETTER JA
	0x091D: "jh",          // DEVANAGARI LETTER JHA
	0x091E: "ny",          // DEVANAGARI LETTER NYA
	0x091F: "tt",          // DEVANAGARI LETTER TTA
	0x0920: "tth",         // DEVANAGARI LETTER TTHA
	0x0921: "dd",          // DEVANAGARI LETTER DDA
	0x0922: "ddh",         // DEVANAGARI LETTER DDHA
	0x0923: "nn",          // DEVANAGARI LETTER NNA
	0x0924: "t",           // DEVANAGARI LETTER TA
	0x0925: "th",          // DEVANAGARI LETTER THA
	0x0926: "d",           // DEVANAGARI LETTER DA
	0x0927: "dh",          // DEVANAGARI LETTER DHA
	0x0928: "n",           // DEVANAGARI LETTER NA
	0x0929: "nnn",         // DEVANAGARI LETTER NNNA
	0x092A: "p",           // DEVANAGARI LETTER PA
	0x092B: "ph",          // DEVANAGARI LETTER PHA
	0x092C: "b",           // DEVANAGARI LETTER BA
	0x092D: "bh",          // DEVANAGARI LETTER BHA
	0x092E: "m",           // DEVANAGARI LETTER MA
	0x092F: "y",           // DEVANAGARI LETTER YA
	0x0930: "r",           // DEVANAGARI LETTER RA
	0x0931: "rr",          // DEVANAGARI LETTER RRA
	0x0932: "l",           // DEVANAGARI LETTER LA
	0x0933: "l",           // DEVANAGARI LETTER LLA
	0x0934: "lll",         // DEVANAGARI LETTER LLLA
	0x0935: "v",           // DEVANAGARI LETTER VA
	0x0936: "sh",          // DEVANAGARI LETTER SHA
	0x0937: "ss",          // DEVANAGARI LETTER SSA
	0x0938: "s",           // DEVANAGARI LETTER SA
	0x0939: "h",           // DEVANAGARI LETTER HA
	0x093C: "'",           // DEVANAGARI SIGN NUKTA
	0x093D: "'",           // DEVANAGARI SIGN AVAGRAHA
	0x093E: "aa",          // DEVANAGARI VOWEL SIGN AA
	0x093F: "i",           // DEVANAGARI VOWEL SIGN I
	0x0940: "ii",          // DEVANAGARI VOWEL SIGN II
	0x0941: "u",           // DEVANAGARI VOWEL SIGN U
	0x0942: "uu",          // DEVANAGARI VOWEL SIGN UU
	0x0943: "R",           // DEVANAGARI VOWEL SIGN VOCALIC R
	0x0944: "RR",          // DEVANAGARI VOWEL SIGN VOCALIC RR
	0x0945: "eN",          // DEVANAGARI VOWEL SIGN CANDRA E
	0x0946: "e",           // DEVANAGARI VOWEL SIGN SHORT E
	0x0947: "e",           // DEVANAGARI VOWEL SIGN E
	0x0948: "ai",          // DEVANAGARI VOWEL SIGN AI
	0x0949: "oN",          // DEVANAGARI VOWEL SIGN CANDRA O
	0x094A: "o",           // DEVANAGARI VOWEL SIGN SHORT O
	0x094B: "o",           // DEVANAGARI VOWEL SIGN O
	0x094C: "au",          // DEVANAGARI VOWEL SIGN AU
	0x0950: "AUM",         // DEVANAGARI OM
	0x0951: "'",           // DEVANAGARI STRESS SIGN UDATTA
	0x0952: "'",           // DEVANAGARI STRESS SIGN ANUDATTA
	0x0953: "`",           // DEVANAGARI GRAVE ACCENT
	0x0954: "'",           // DEVANAGARI ACUTE ACCENT
	0x0958: "q",           // DEVANAGARI LETTER QA
	0x0959: "khh",         // DEVANAGARI LETTER KHHA
	0x095A: "ghh",         // DEVANAGARI LETTER GHHA
	0x095B: "z",           // DEVANAGARI LETTER ZA
	0x095C: "dddh",        // DEVANAGARI LETTER DDDHA
	0x095D: "rh",          // DEVANAGARI LETTER RHA
	0x095E: "f",           // DEVANAGARI LETTER FA
	0x095F: "yy",          // DEVANAGARI LETTER YYA
	0x0960: "RR",          // DEVANAGARI LETTER VOCALIC RR
	0x0961: "LL",          // DEVANAGARI LETTER VOCALIC LL
	0x0962: "L",           // DEVANAGARI VOWEL SIGN VOCALIC L
	0x0963: "LL",          // DEVANAGARI VOWEL SIGN VOCALIC LL
	0x0964: " / ",         // DEVANAGARI DANDA
	0x0965: " // ",        // DEVANAGARI DOUBLE DANDA
	0x0966: "0",           // DEVANAGARI DIGIT ZERO
	0x0967: "1",           // DEVANAGARI DIGIT ONE
	0x0968: "2",           // DEVANAGARI DIGIT TWO
	0x0969: "3",           // DEVANAGARI DIGIT THREE
	0x096A: "4",           // DEVANAGARI DIGIT FOUR
	0x096B: "5",           // DEVANAGARI DIGIT FIVE
	0x096C: "6",           // DEVANAGARI DIGIT SIX
	0x096D: "7",           // DEVANAGARI DIGIT SEVEN
	0x096E: "8",           // DEVANAGARI DIGIT EIGHT
	0x096F: "9",           // DEVANAGARI DIGIT NINE
	0x0970: ".",           // DEVANAGARI ABBREVIATION SIGN
	0x0981: "N",           // BENGALI SIGN CANDRABINDU
	0x0982: "N",           // BENGALI SIGN ANUSVARA
	0x0983: "H",           // BENGALI SIGN VISARGA
	0x0985: "a",           // BENGALI LETTER A
	0x0986: "aa",          // BENGALI LETTER AA
	0x0987: "i",           // BENGALI LETTER I
	0x0988: "ii",          // BENGALI LETTER II
	0x0989: "u",           // BENGALI LETTER U
	0x098A: "uu",          // BENGALI LETTER UU
	0x098B: "R",           // BENGALI LETTER VOCALIC R
	0x098C: "RR",          // BENGALI LETTER VOCALIC L
	0x098F: "e",           // BENGALI LETTER E
	0x0990: "ai",          // BENGALI LETTER AI
	0x0993: "o",           // BENGALI LETTER O
	0x0994: "au",          // BENGALI LETTER AU
	0x0995: "k",           // BENGALI LETTER KA
	0x0996: "kh",          // BENGALI LETTER KHA
	0x0997: "g",           // BENGALI LETTER GA
	0x0998: "gh",          // BENGALI LETTER GHA
	0x0999: "ng",          // BENGALI LETTER NGA
	0x099A: "c",           // BENGALI LETTER CA
	0x099B: "ch",          // BENGALI LETTER CHA
	0x099C: "j",           // BENGALI LETTER JA
	0x099D: "jh",          // BENGALI LETTER JHA
	0x099E: "ny",          // BENGALI LETTER NYA
	0x099F: "tt",          // BENGALI LETTER TTA
	0x09A0: "tth",         // BENGALI LETTER TTHA
	0x09A1: "dd",          // BENGALI LETTER DDA
	0x09A2: "ddh",         // BENGALI LETTER DDHA
	0x09A3: "nn",          // BENGALI LETTER NNA
	0x09A4: "t",           // BENGALI LETTER TA
	0x09A5: "th",          // BENGALI LETTER THA
	0x09A6: "d",           // BENGALI LETTER DA
	0x09A7: "dh",          // BENGALI LETTER DHA
	0x09A8: "n",           // BENGALI LETTER NA
	0x09AA: "p",           // BENGALI LETTER PA
	0x09AB: "ph",          // BENGALI LETTER PHA
	0x09AC: "b",           // BENGALI LETTER BA
	0x09AD: "bh",          // BENGALI LETTER BHA
	0x09AE: "m",           // BENGALI LETTER MA
	0x09AF: "y",           // BENGALI LETTER YA
	0x09B0: "r",           // BENGALI LETTER RA
	0x09B2: "l",           // BENGALI LETTER LA
	0x09B6: "sh",          // BENGALI LETTER SHA
	0x09B7: "ss",          // BENGALI LETTER SSA
	0x09B8: "s",           // BENGALI LETTER SA
	0x09B9: "h",           // BENGALI LETTER HA
	0x09BC: "'",           // BENGALI SIGN NUKTA
	0x09BE: "aa",          // BENGALI VOWEL SIGN AA
	0x09BF: "i",           // BENGALI VOWEL SIGN I
	0x09C0: "ii",          // BENGALI VOWEL SIGN II
	0x09C1: "u",           // BENGALI VOWEL SIGN U
	0x09C2: "uu",          // BENGALI VOWEL SIGN UU
	0x09C3: "R",           // BENGALI VOWEL SIGN VOCALIC R
	0x09C4: "RR",          // BENGALI VOWEL SIGN VOCALIC RR
	0x09C7: "e",           // BENGALI VOWEL SIGN E
	0x09C8: "ai",          // BENGALI VOWEL SIGN AI
	0x09CB: "o",           // BENGALI VOWEL SIGN O
	0x09CC: "au",          // BENGALI VOWEL SIGN AU
	0x09D7: "+",           // BENGALI AU LENGTH MARK
	0x09DC: "rr",          // BENGALI LETTER RRA
	0x09DD: "rh",          // BENGALI LETTER RHA
	0x09DF: "yy",          // BENGALI LETTER YYA
	0x09E0: "RR",          // BENGALI LETTER VOCALIC RR
	0x09E1: "LL",          // BENGALI LETTER VOCALIC LL
	0x09E2: "L",           // BENGALI VOWEL SIGN VOCALIC L
	0x09E3: "LL",          // BENGALI VOWEL SIGN VOCALIC LL
	0x09E6: "0",           // BENGALI DIGIT ZERO
	0x09E7: "1",           // BENGALI DIGIT ONE
	0x09E8: "2",           // BENGALI DIGIT TWO
	0x09E9: "3",           // BENGALI DIGIT THREE
	0x09EA: "4",           // BENGALI DIGIT FOUR
	0x09EB: "5",           // BENGALI DIGIT FIVE
	0x09EC: "6",           // BENGALI DIGIT SIX
	0x09ED: "7",           // BENGALI DIGIT SEVEN
	0x09EE: "8",           // BENGALI DIGIT EIGHT
	0x09EF: "9",           // BENGALI DIGIT NINE
	0x09F0: "r'",          // BENGALI LETTER RA WITH MIDDLE DIAGONAL
	0x09F1: "r`",          // BENGALI LETTER RA WITH LOWER DIAGONAL
	0x09F2: "Rs",          // BENGALI RUPEE MARK
	0x09F3: "Rs",          // BENGALI RUPEE SIGN
	0x09F4: "1/",          // BENGALI CURRENCY NUMERATOR ONE
	0x09F5: "2/",          // BENGALI CURRENCY NUMERATOR TWO
	0x09F6: "3/",          // BENGALI CURRENCY NUMERATOR THREE
	0x09F7: "4/",          // BENGALI CURRENCY NUMERATOR FOUR
	0x09F8: " 1 - 1/",     // BENGALI CURRENCY NUMERATOR ONE LESS THAN THE DENOMINATOR
	0x09F9: "/16",         // BENGALI CURRENCY DENOMINATOR SIXTEEN
	0x0A02: "N",           // GURMUKHI SIGN BINDI
	0x0A05: "a",           // GURMUKHI LETTER A
	0x0A06: "aa",          // GURMUKHI LETTER AA
	0x0A07: "i",           // GURMUKHI LETTER I
	0x0A08: "ii",          // GURMUKHI LETTER II
	0x0A09: "u",           // GURMUKHI LETTER U
	0x0A0A: "uu",          // GURMUKHI LETTER UU
	0x0A0F: "ee",          // GURMUKHI LETTER EE
	0x0A10: "ai",          // GURMUKHI LETTER AI
	0x0A13: "oo",          // GURMUKHI LETTER OO
	0x0A14: "au",          // GURMUKHI LETTER AU
	0x0A15: "k",           // GURMUKHI LETTER KA
	0x0A16: "kh",          // GURMUKHI LETTER KHA
	0x0A17: "g",           // GURMUKHI LETTER GA
	0x0A18: "gh",          // GURMUKHI LETTER GHA
	0x0A19: "ng",          // GURMUKHI LETTER NGA
	0x0A1A: "c",           // GURMUKHI LETTER CA
	0x0A1B: "ch",          // GURMUKHI LETTER CHA
	0x0A1C: "j",           // GURMUKHI LETTER JA
	0x0A1D: "jh",          // GURMUKHI LETTER JHA
	0x0A1E: "ny",          // GURMUKHI LETTER NYA
	0x0A1F: "tt",          // GURMUKHI LETTER TTA
	0x0A20: "tth",         // GURMUKHI LETTER TTHA
	0x0A21: "dd",          // GURMUKHI LETTER DDA
	0x0A22: "ddh",         // GURMUKHI LETTER DDHA
	0x0A23: "nn",          // GURMUKHI LETTER NNA
	0x0A24: "t",           // GURMUKHI LETTER TA
	0x0A25: "th",          // GURMUKHI LETTER THA
	0x0A26: "d",           // GURMUKHI LETTER DA
	0x0A27: "dh",          // GURMUKHI LETTER DHA
	0x0A28: "n",           // GURMUKHI LETTER NA
	0x0A2A: "p",           // GURMUKHI LETTER PA
	0x0A2B: "ph",          // GURMUKHI LETTER PHA
	0x0A2C: "b",           // GURMUKHI LETTER BA
	0x0A2D: "bb",          // GURMUKHI LETTER BHA
	0x0A2E: "m",           // GURMUKHI LETTER MA
	0x0A2F: "y",           // GURMUKHI LETTER YA
	0x0A30: "r",           // GURMUKHI LETTER RA
	0x0A32: "l",           // GURMUKHI LETTER LA
	0x0A33: "ll",          // GURMUKHI LETTER LLA
	0x0A35: "v",           // GURMUKHI LETTER VA
	0x0A36: "sh",          // GURMUKHI LETTER SHA
	0x0A38: "s",           // GURMUKHI LETTER SA
	0x0A39: "h",           // GURMUKHI LETTER HA
	0x0A3C: "'",           // GURMUKHI SIGN NUKTA
	0x0A3E: "aa",          // GURMUKHI VOWEL SIGN AA
	0x0A3F: "i",           // GURMUKHI VOWEL SIGN I
	0x0A40: "ii",          // GURMUKHI VOWEL SIGN II
	0x0A41: "u",           // GURMUKHI VOWEL SIGN U
	0x0A42: "uu",          // GURMUKHI VOWEL SIGN UU
	0x0A47: "ee",          // GURMUKHI VOWEL SIGN EE
	0x0A48: "ai",          // GURMUKHI VOWEL SIGN AI
	0x0A4B: "oo",          // GURMUKHI VOWEL SIGN OO
	0x0A4C: "au",          // GURMUKHI VOWEL SIGN AU
	0x0A59: "khh",         // GURMUKHI LETTER KHHA
	0x0A5A: "ghh",         // GURMUKHI LETTER GHHA
	0x0A5B: "z",           // GURMUKHI LETTER ZA
	0x0A5C: "rr",          // GURMUKHI LETTER RRA
	0x0A5E: "f",           // GURMUKHI LETTER FA
	0x0A66: "0",           // GURMUKHI DIGIT ZERO
	0x0A67: "1",           // GURMUKHI DIGIT ONE
	0x0A68: "2",           // GURMUKHI DIGIT TWO
	0x0A69: "3",           // GURMUKHI DIGIT THREE
	0x0A6A: "4",           // GURMUKHI DIGIT FOUR
	0x0A6B: "5",           // GURMUKHI DIGIT FIVE
	0x0A6C: "6",           // GURMUKHI DIGIT SIX
	0x0A6D: "7",           // GURMUKHI DIGIT SEVEN
	0x0A6E: "8",           // GURMUKHI DIGIT EIGHT
	0x0A6F: "9",           // GURMUKHI DIGIT NINE
	0x0A70: "N",           // GURMUKHI TIPPI
	0x0A71: "H",           // GURMUKHI ADDAK
	0x0A74: "G.E.O.",      // GURMUKHI EK ONKAR
	0x0A81: "N",           // GUJARATI SIGN CANDRABINDU
	0x0A82: "N",           // GUJARATI SIGN ANUSVARA
	0x0A83: "H",           // GUJARATI SIGN VISARGA
	0x0A85: "a",           // GUJARATI LETTER A
	0x0A86: "aa",          // GUJARATI LETTER AA
	0x0A87: "i",           // GUJARATI LETTER I
	0x0A88: "ii",          // GUJARATI LETTER II
	0x0A89: "u",           // GUJARATI LETTER U
	0x0A8A: "uu",          // GUJARATI LETTER UU
	0x0A8B: "R",           // GUJARATI LETTER VOCALIC R
	0x0A8C: "",            //
	0x0A8D: "eN",          // GUJARATI VOWEL CANDRA E
	0x0A8F: "e",           // GUJARATI LETTER E
	0x0A90: "ai",          // GUJARATI LETTER AI
	0x0A91: "oN",          // GUJARATI VOWEL CANDRA O
	0x0A93: "o",           // GUJARATI LETTER O
	0x0A94: "au",          // GUJARATI LETTER AU
	0x0A95: "k",           // GUJARATI LETTER KA
	0x0A96: "kh",          // GUJARATI LETTER KHA
	0x0A97: "g",           // GUJARATI LETTER GA
	0x0A98: "gh",          // GUJARATI LETTER GHA
	0x0A99: "ng",          // GUJARATI LETTER NGA
	0x0A9A: "c",           // GUJARATI LETTER CA
	0x0A9B: "ch",          // GUJARATI LETTER CHA
	0x0A9C: "j",           // GUJARATI LETTER JA
	0x0A9D: "jh",          // GUJARATI LETTER JHA
	0x0A9E: "ny",          // GUJARATI LETTER NYA
	0x0A9F: "tt",          // GUJARATI LETTER TTA
	0x0AA0: "tth",         // GUJARATI LETTER TTHA
	0x0AA1: "dd",          // GUJARATI LETTER DDA
	0x0AA2: "ddh",         // GUJARATI LETTER DDHA
	0x0AA3: "nn",          // GUJARATI LETTER NNA
	0x0AA4: "t",           // GUJARATI LETTER TA
	0x0AA5: "th",          // GUJARATI LETTER THA
	0x0AA6: "d",           // GUJARATI LETTER DA
	0x0AA7: "dh",          // GUJARATI LETTER DHA
	0x0AA8: "n",           // GUJARATI LETTER NA
	0x0AAA: "p",           // GUJARATI LETTER PA
	0x0AAB: "ph",          // GUJARATI LETTER PHA
	0x0AAC: "b",           // GUJARATI LETTER BA
	0x0AAD: "bh",          // GUJARATI LETTER BHA
	0x0AAE: "m",           // GUJARATI LETTER MA
	0x0AAF: "ya",          // GUJARATI LETTER YA
	0x0AB0: "r",           // GUJARATI LETTER RA
	0x0AB2: "l",           // GUJARATI LETTER LA
	0x0AB3: "ll",          // GUJARATI LETTER LLA
	0x0AB5: "v",           // GUJARATI LETTER VA
	0x0AB6: "sh",          // GUJARATI LETTER SHA
	0x0AB7: "ss",          // GUJARATI LETTER SSA
	0x0AB8: "s",           // GUJARATI LETTER SA
	0x0AB9: "h",           // GUJARATI LETTER HA
	0x0ABC: "'",           // GUJARATI SIGN NUKTA
	0x0ABD: "'",           // GUJARATI SIGN AVAGRAHA
	0x0ABE: "aa",          // GUJARATI VOWEL SIGN AA
	0x0ABF: "i",           // GUJARATI VOWEL SIGN I
	0x0AC0: "ii",          // GUJARATI VOWEL SIGN II
	0x0AC1: "u",           // GUJARATI VOWEL SIGN U
	0x0AC2: "uu",          // GUJARATI VOWEL SIGN UU
	0x0AC3: "R",           // GUJARATI VOWEL SIGN VOCALIC R
	0x0AC4: "RR",          // GUJARATI VOWEL SIGN VOCALIC RR
	0x0AC5: "eN",          // GUJARATI VOWEL SIGN CANDRA E
	0x0AC7: "e",           // GUJARATI VOWEL SIGN E
	0x0AC8: "ai",          // GUJARATI VOWEL SIGN AI
	0x0AC9: "oN",          // GUJARATI VOWEL SIGN CANDRA O
	0x0ACB: "o",           // GUJARATI VOWEL SIGN O
	0x0ACC: "au",          // GUJARATI VOWEL SIGN AU
	0x0AD0: "AUM",         // GUJARATI OM
	0x0AE0: "RR",          // GUJARATI LETTER VOCALIC RR
	0x0AE6: "0",           // GUJARATI DIGIT ZERO
	0x0AE7: "1",           // GUJARATI DIGIT ONE
	0x0AE8: "2",           // GUJARATI DIGIT TWO
	0x0AE9: "3",           // GUJARATI DIGIT THREE
	0x0AEA: "4",           // GUJARATI DIGIT FOUR
	0x0AEB: "5",           // GUJARATI DIGIT FIVE
	0x0AEC: "6",           // GUJARATI DIGIT SIX
	0x0AED: "7",           // GUJARATI DIGIT SEVEN
	0x0AEE: "8",           // GUJARATI DIGIT EIGHT
	0x0AEF: "9",           // GUJARATI DIGIT NINE
	0x0B01: "N",           // ORIYA SIGN CANDRABINDU
	0x0B02: "N",           // ORIYA SIGN ANUSVARA
	0x0B03: "H",           // ORIYA SIGN VISARGA
	0x0B05: "a",           // ORIYA LETTER A
	0x0B06: "aa",          // ORIYA LETTER AA
	0x0B07: "i",           // ORIYA LETTER I
	0x0B08: "ii",          // ORIYA LETTER II
	0x0B09: "u",           // ORIYA LETTER U
	0x0B0A: "uu",          // ORIYA LETTER UU
	0x0B0B: "R",           // ORIYA LETTER VOCALIC R
	0x0B0C: "L",           // ORIYA LETTER VOCALIC L
	0x0B0F: "e",           // ORIYA LETTER E
	0x0B10: "ai",          // ORIYA LETTER AI
	0x0B13: "o",           // ORIYA LETTER O
	0x0B14: "au",          // ORIYA LETTER AU
	0x0B15: "k",           // ORIYA LETTER KA
	0x0B16: "kh",          // ORIYA LETTER KHA
	0x0B17: "g",           // ORIYA LETTER GA
	0x0B18: "gh",          // ORIYA LETTER GHA
	0x0B19: "ng",          // ORIYA LETTER NGA
	0x0B1A: "c",           // ORIYA LETTER CA
	0x0B1B: "ch",          // ORIYA LETTER CHA
	0x0B1C: "j",           // ORIYA LETTER JA
	0x0B1D: "jh",          // ORIYA LETTER JHA
	0x0B1E: "ny",          // ORIYA LETTER NYA
	0x0B1F: "tt",          // ORIYA LETTER TTA
	0x0B20: "tth",         // ORIYA LETTER TTHA
	0x0B21: "dd",          // ORIYA LETTER DDA
	0x0B22: "ddh",         // ORIYA LETTER DDHA
	0x0B23: "nn",          // ORIYA LETTER NNA
	0x0B24: "t",           // ORIYA LETTER TA
	0x0B25: "th",          // ORIYA LETTER THA
	0x0B26: "d",           // ORIYA LETTER DA
	0x0B27: "dh",          // ORIYA LETTER DHA
	0x0B28: "n",           // ORIYA LETTER NA
	0x0B2A: "p",           // ORIYA LETTER PA
	0x0B2B: "ph",          // ORIYA LETTER PHA
	0x0B2C: "b",           // ORIYA LETTER BA
	0x0B2D: "bh",          // ORIYA LETTER BHA
	0x0B2E: "m",           // ORIYA LETTER MA
	0x0B2F: "y",           // ORIYA LETTER YA
	0x0B30: "r",           // ORIYA LETTER RA
	0x0B32: "l",           // ORIYA LETTER LA
	0x0B33: "ll",          // ORIYA LETTER LLA
	0x0B36: "sh",          // ORIYA LETTER SHA
	0x0B37: "ss",          // ORIYA LETTER SSA
	0x0B38: "s",           // ORIYA LETTER SA
	0x0B39: "h",           // ORIYA LETTER HA
	0x0B3C: "'",           // ORIYA SIGN NUKTA
	0x0B3D: "'",           // ORIYA SIGN AVAGRAHA
	0x0B3E: "aa",          // ORIYA VOWEL SIGN AA
	0x0B3F: "i",           // ORIYA VOWEL SIGN I
	0x0B40: "ii",          // ORIYA VOWEL SIGN II
	0x0B41: "u",           // ORIYA VOWEL SIGN U
	0x0B42: "uu",          // ORIYA VOWEL SIGN UU
	0x0B43: "R",           // ORIYA VOWEL SIGN VOCALIC R
	0x0B47: "e",           // ORIYA VOWEL SIGN E
	0x0B48: "ai",          // ORIYA VOWEL SIGN AI
	0x0B4B: "o",           // ORIYA VOWEL SIGN O
	0x0B4C: "au",          // ORIYA VOWEL SIGN AU
	0x0B56: "+",           // ORIYA AI LENGTH MARK
	0x0B57: "+",           // ORIYA AU LENGTH MARK
	0x0B5C: "rr",          // ORIYA LETTER RRA
	0x0B5D: "rh",          // ORIYA LETTER RHA
	0x0B5F: "yy",          // ORIYA LETTER YYA
	0x0B60: "RR",          // ORIYA LETTER VOCALIC RR
	0x0B61: "LL",          // ORIYA LETTER VOCALIC LL
	0x0B66: "0",           // ORIYA DIGIT ZERO
	0x0B67: "1",           // ORIYA DIGIT ONE
	0x0B68: "2",           // ORIYA DIGIT TWO
	0x0B69: "3",           // ORIYA DIGIT THREE
	0x0B6A: "4",           // ORIYA DIGIT FOUR
	0x0B6B: "5",           // ORIYA DIGIT FIVE
	0x0B6C: "6",           // ORIYA DIGIT SIX
	0x0B6D: "7",           // ORIYA DIGIT SEVEN
	0x0B6E: "8",           // ORIYA DIGIT EIGHT
	0x0B6F: "9",           // ORIYA DIGIT NINE
	0x0B82: "N",           // TAMIL SIGN ANUSVARA
	0x0B83: "H",           // TAMIL SIGN VISARGA
	0x0B85: "a",           // TAMIL LETTER A
	0x0B86: "aa",          // TAMIL LETTER AA
	0x0B87: "i",           // TAMIL LETTER I
	0x0B88: "ii",          // TAMIL LETTER II
	0x0B89: "u",           // TAMIL LETTER U
	0x0B8A: "uu",          // TAMIL LETTER UU
	0x0B8E: "e",           // TAMIL LETTER E
	0x0B8F: "ee",          // TAMIL LETTER EE
	0x0B90: "ai",          // TAMIL LETTER AI
	0x0B92: "o",           // TAMIL LETTER O
	0x0B93: "oo",          // TAMIL LETTER OO
	0x0B94: "au",          // TAMIL LETTER AU
	0x0B95: "k",           // TAMIL LETTER KA
	0x0B99: "ng",          // TAMIL LETTER NGA
	0x0B9A: "c",           // TAMIL LETTER CA
	0x0B9C: "j",           // TAMIL LETTER JA
	0x0B9E: "ny",          // TAMIL LETTER NYA
	0x0B9F: "tt",          // TAMIL LETTER TTA
	0x0BA3: "nn",          // TAMIL LETTER NNA
	0x0BA4: "t",           // TAMIL LETTER TA
	0x0BA8: "n",           // TAMIL LETTER NA
	0x0BA9: "nnn",         // TAMIL LETTER NNNA
	0x0BAA: "p",           // TAMIL LETTER PA
	0x0BAE: "m",           // TAMIL LETTER MA
	0x0BAF: "y",           // TAMIL LETTER YA
	0x0BB0: "r",           // TAMIL LETTER RA
	0x0BB1: "rr",          // TAMIL LETTER RRA
	0x0BB2: "l",           // TAMIL LETTER LA
	0x0BB3: "ll",          // TAMIL LETTER LLA
	0x0BB4: "lll",         // TAMIL LETTER LLLA
	0x0BB5: "v",           // TAMIL LETTER VA
	0x0BB6: "",            //
	0x0BB7: "ss",          // TAMIL LETTER SSA
	0x0BB8: "s",           // TAMIL LETTER SA
	0x0BB9: "h",           // TAMIL LETTER HA
	0x0BBE: "aa",          // TAMIL VOWEL SIGN AA
	0x0BBF: "i",           // TAMIL VOWEL SIGN I
	0x0BC0: "ii",          // TAMIL VOWEL SIGN II
	0x0BC1: "u",           // TAMIL VOWEL SIGN U
	0x0BC2: "uu",          // TAMIL VOWEL SIGN UU
	0x0BC6: "e",           // TAMIL VOWEL SIGN E
	0x0BC7: "ee",          // TAMIL VOWEL SIGN EE
	0x0BC8: "ai",          // TAMIL VOWEL SIGN AI
	0x0BCA: "o",           // TAMIL VOWEL SIGN O
	0x0BCB: "oo",          // TAMIL VOWEL SIGN OO
	0x0BCC: "au",          // TAMIL VOWEL SIGN AU
	0x0BD7: "+",           // TAMIL AU LENGTH MARK
	0x0BE6: "0",           //
	0x0BE7: "1",           // TAMIL DIGIT ONE
	0x0BE8: "2",           // TAMIL DIGIT TWO
	0x0BE9: "3",           // TAMIL DIGIT THREE
	0x0BEA: "4",           // TAMIL DIGIT FOUR
	0x0BEB: "5",           // TAMIL DIGIT FIVE
	0x0BEC: "6",           // TAMIL DIGIT SIX
	0x0BED: "7",           // TAMIL DIGIT SEVEN
	0x0BEE: "8",           // TAMIL DIGIT EIGHT
	0x0BEF: "9",           // TAMIL DIGIT NINE
	0x0BF0: "+10+",        // TAMIL NUMBER TEN
	0x0BF1: "+100+",       // TAMIL NUMBER ONE HUNDRED
	0x0BF2: "+1000+",      // TAMIL NUMBER ONE THOUSAND
	0x0C01: "N",           // TELUGU SIGN CANDRABINDU
	0x0C02: "N",           // TELUGU SIGN ANUSVARA
	0x0C03: "H",           // TELUGU SIGN VISARGA
	0x0C05: "a",           // TELUGU LETTER A
	0x0C06: "aa",          // TELUGU LETTER AA
	0x0C07: "i",           // TELUGU LETTER I
	0x0C08: "ii",          // TELUGU LETTER II
	0x0C09: "u",           // TELUGU LETTER U
	0x0C0A: "uu",          // TELUGU LETTER UU
	0x0C0B: "R",           // TELUGU LETTER VOCALIC R
	0x0C0C: "L",           // TELUGU LETTER VOCALIC L
	0x0C0E: "e",           // TELUGU LETTER E
	0x0C0F: "ee",          // TELUGU LETTER EE
	0x0C10: "ai",          // TELUGU LETTER AI
	0x0C12: "o",           // TELUGU LETTER O
	0x0C13: "oo",          // TELUGU LETTER OO
	0x0C14: "au",          // TELUGU LETTER AU
	0x0C15: "k",           // TELUGU LETTER KA
	0x0C16: "kh",          // TELUGU LETTER KHA
	0x0C17: "g",           // TELUGU LETTER GA
	0x0C18: "gh",          // TELUGU LETTER GHA
	0x0C19: "ng",          // TELUGU LETTER NGA
	0x0C1A: "c",           // TELUGU LETTER CA
	0x0C1B: "ch",          // TELUGU LETTER CHA
	0x0C1C: "j",           // TELUGU LETTER JA
	0x0C1D: "jh",          // TELUGU LETTER JHA
	0x0C1E: "ny",          // TELUGU LETTER NYA
	0x0C1F: "tt",          // TELUGU LETTER TTA
	0x0C20: "tth",         // TELUGU LETTER TTHA
	0x0C21: "dd",          // TELUGU LETTER DDA
	0x0C22: "ddh",         // TELUGU LETTER DDHA
	0x0C23: "nn",          // TELUGU LETTER NNA
	0x0C24: "t",           // TELUGU LETTER TA
	0x0C25: "th",          // TELUGU LETTER THA
	0x0C26: "d",           // TELUGU LETTER DA
	0x0C27: "dh",          // TELUGU LETTER DHA
	0x0C28: "n",           // TELUGU LETTER NA
	0x0C2A: "p",           // TELUGU LETTER PA
	0x0C2B: "ph",          // TELUGU LETTER PHA
	0x0C2C: "b",           // TELUGU LETTER BA
	0x0C2D: "bh",          // TELUGU LETTER BHA
	0x0C2E: "m",           // TELUGU LETTER MA
	0x0C2F: "y",           // TELUGU LETTER YA
	0x0C30: "r",           // TELUGU LETTER RA
	0x0C31: "rr",          // TELUGU LETTER RRA
	0x0C32: "l",           // TELUGU LETTER LA
	0x0C33: "ll",          // TELUGU LETTER LLA
	0x0C35: "v",           // TELUGU LETTER VA
	0x0C36: "sh",          // TELUGU LETTER SHA
	0x0C37: "ss",          // TELUGU LETTER SSA
	0x0C38: "s",           // TELUGU LETTER SA
	0x0C39: "h",           // TELUGU LETTER HA
	0x0C3E: "aa",          // TELUGU VOWEL SIGN AA
	0x0C3F: "i",           // TELUGU VOWEL SIGN I
	0x0C40: "ii",          // TELUGU VOWEL SIGN II
	0x0C41: "u",           // TELUGU VOWEL SIGN U
	0x0C42: "uu",          // TELUGU VOWEL SIGN UU
	0x0C43: "R",           // TELUGU VOWEL SIGN VOCALIC R
	0x0C44: "RR",          // TELUGU VOWEL SIGN VOCALIC RR
	0x0C46: "e",           // TELUGU VOWEL SIGN E
	0x0C47: "ee",          // TELUGU VOWEL SIGN EE
	0x0C48: "ai",          // TELUGU VOWEL SIGN AI
	0x0C4A: "o",           // TELUGU VOWEL SIGN O
	0x0C4B: "oo",          // TELUGU VOWEL SIGN OO
	0x0C4C: "au",          // TELUGU VOWEL SIGN AU
	0x0C55: "+",           // TELUGU LENGTH MARK
	0x0C56: "+",           // TELUGU AI LENGTH MARK
	0x0C60: "RR",          // TELUGU LETTER VOCALIC RR
	0x0C61: "LL",          // TELUGU LETTER VOCALIC LL
	0x0C66: "0",           // TELUGU DIGIT ZERO
	0x0C67: "1",           // TELUGU DIGIT ONE
	0x0C68: "2",           // TELUGU DIGIT TWO
	0x0C69: "3",           // TELUGU DIGIT THREE
	0x0C6A: "4",           // TELUGU DIGIT FOUR
	0x0C6B: "5",           // TELUGU DIGIT FIVE
	0x0C6C: "6",           // TELUGU DIGIT SIX
	0x0C6D: "7",           // TELUGU DIGIT SEVEN
	0x0C6E: "8",           // TELUGU DIGIT EIGHT
	0x0C6F: "9",           // TELUGU DIGIT NINE
	0x0C82: "N",           // KANNADA SIGN ANUSVARA
	0x0C83: "H",           // KANNADA SIGN VISARGA
	0x0C85: "a",           // KANNADA LETTER A
	0x0C86: "aa",          // KANNADA LETTER AA
	0x0C87: "i",           // KANNADA LETTER I
	0x0C88: "ii",          // KANNADA LETTER II
	0x0C89: "u",           // KANNADA LETTER U
	0x0C8A: "uu",          // KANNADA LETTER UU
	0x0C8B: "R",           // KANNADA LETTER VOCALIC R
	0x0C8C: "L",           // KANNADA LETTER VOCALIC L
	0x0C8E: "e",           // KANNADA LETTER E
	0x0C8F: "ee",          // KANNADA LETTER EE
	0x0C90: "ai",          // KANNADA LETTER AI
	0x0C92: "o",           // KANNADA LETTER O
	0x0C93: "oo",          // KANNADA LETTER OO
	0x0C94: "au",          // KANNADA LETTER AU
	0x0C95: "k",           // KANNADA LETTER KA
	0x0C96: "kh",          // KANNADA LETTER KHA
	0x0C97: "g",           // KANNADA LETTER GA
	0x0C98: "gh",          // KANNADA LETTER GHA
	0x0C99: "ng",          // KANNADA LETTER NGA
	0x0C9A: "c",           // KANNADA LETTER CA
	0x0C9B: "ch",          // KANNADA LETTER CHA
	0x0C9C: "j",           // KANNADA LETTER JA
	0x0C9D: "jh",          // KANNADA LETTER JHA
	0x0C9E: "ny",          // KANNADA LETTER NYA
	0x0C9F: "tt",          // KANNADA LETTER TTA
	0x0CA0: "tth",         // KANNADA LETTER TTHA
	0x0CA1: "dd",          // KANNADA LETTER DDA
	0x0CA2: "ddh",         // KANNADA LETTER DDHA
	0x0CA3: "nn",          // KANNADA LETTER NNA
	0x0CA4: "t",           // KANNADA LETTER TA
	0x0CA5: "th",          // KANNADA LETTER THA
	0x0CA6: "d",           // KANNADA LETTER DA
	0x0CA7: "dh",          // KANNADA LETTER DHA
	0x0CA8: "n",           // KANNADA LETTER NA
	0x0CAA: "p",           // KANNADA LETTER PA
	0x0CAB: "ph",          // KANNADA LETTER PHA
	0x0CAC: "b",           // KANNADA LETTER BA
	0x0CAD: "bh",          // KANNADA LETTER BHA
	0x0CAE: "m",           // KANNADA LETTER MA
	0x0CAF: "y",           // KANNADA LETTER YA
	0x0CB0: "r",           // KANNADA LETTER RA
	0x0CB1: "rr",          // KANNADA LETTER RRA
	0x0CB2: "l",           // KANNADA LETTER LA
	0x0CB3: "ll",          // KANNADA LETTER LLA
	0x0CB5: "v",           // KANNADA LETTER VA
	0x0CB6: "sh",          // KANNADA LETTER SHA
	0x0CB7: "ss",          // KANNADA LETTER SSA
	0x0CB8: "s",           // KANNADA LETTER SA
	0x0CB9: "h",           // KANNADA LETTER HA
	0x0CBE: "aa",          // KANNADA VOWEL SIGN AA
	0x0CBF: "i",           // KANNADA VOWEL SIGN I
	0x0CC0: "ii",          // KANNADA VOWEL SIGN II
	0x0CC1: "u",           // KANNADA VOWEL SIGN U
	0x0CC2: "uu",          // KANNADA VOWEL SIGN UU
	0x0CC3: "R",           // KANNADA VOWEL SIGN VOCALIC R
	0x0CC4: "RR",          // KANNADA VOWEL SIGN VOCALIC RR
	0x0CC6: "e",           // KANNADA VOWEL SIGN E
	0x0CC7: "ee",          // KANNADA VOWEL SIGN EE
	0x0CC8: "ai",          // KANNADA VOWEL SIGN AI
	0x0CCA: "o",           // KANNADA VOWEL SIGN O
	0x0CCB: "oo",          // KANNADA VOWEL SIGN OO
	0x0CCC: "au",          // KANNADA VOWEL SIGN AU
	0x0CD5: "+",           // KANNADA LENGTH MARK
	0x0CD6: "+",           // KANNADA AI LENGTH MARK
	0x0CDE: "lll",         // KANNADA LETTER FA
	0x0CE0: "RR",          // KANNADA LETTER VOCALIC RR
	0x0CE1: "LL",          // KANNADA LETTER VOCALIC LL
	0x0CE6: "0",           // KANNADA DIGIT ZERO
	0x0CE7: "1",           // KANNADA DIGIT ONE
	0x0CE8: "2",           // KANNADA DIGIT TWO
	0x0CE9: "3",           // KANNADA DIGIT THREE
	0x0CEA: "4",           // KANNADA DIGIT FOUR
	0x0CEB: "5",           // KANNADA DIGIT FIVE
	0x0CEC: "6",           // KANNADA DIGIT SIX
	0x0CED: "7",           // KANNADA DIGIT SEVEN
	0x0CEE: "8",           // KANNADA DIGIT EIGHT
	0x0CEF: "9",           // KANNADA DIGIT NINE
	0x0D02: "N",           // MALAYALAM SIGN ANUSVARA
	0x0D03: "H",           // MALAYALAM SIGN VISARGA
	0x0D05: "a",           // MALAYALAM LETTER A
	0x0D06: "aa",          // MALAYALAM LETTER AA
	0x0D07: "i",           // MALAYALAM LETTER I
	0x0D08: "ii",          // MALAYALAM LETTER II
	0x0D09: "u",           // MALAYALAM LETTER U
	0x0D0A: "uu",          // MALAYALAM LETTER UU
	0x0D0B: "R",           // MALAYALAM LETTER VOCALIC R
	0x0D0C: "L",           // MALAYALAM LETTER VOCALIC L
	0x0D0E: "e",           // MALAYALAM LETTER E
	0x0D0F: "ee",          // MALAYALAM LETTER EE
	0x0D10: "ai",          // MALAYALAM LETTER AI
	0x0D12: "o",           // MALAYALAM LETTER O
	0x0D13: "oo",          // MALAYALAM LETTER OO
	0x0D14: "au",          // MALAYALAM LETTER AU
	0x0D15: "k",           // MALAYALAM LETTER KA
	0x0D16: "kh",          // MALAYALAM LETTER KHA
	0x0D17: "g",           // MALAYALAM LETTER GA
	0x0D18: "gh",          // MALAYALAM LETTER GHA
	0x0D19: "ng",          // MALAYALAM LETTER NGA
	0x0D1A: "c",           // MALAYALAM LETTER CA
	0x0D1B: "ch",          // MALAYALAM LETTER CHA
	0x0D1C: "j",           // MALAYALAM LETTER JA
	0x0D1D: "jh",          // MALAYALAM LETTER JHA
	0x0D1E: "ny",          // MALAYALAM LETTER NYA
	0x0D1F: "tt",          // MALAYALAM LETTER TTA
	0x0D20: "tth",         // MALAYALAM LETTER TTHA
	0x0D21: "dd",          // MALAYALAM LETTER DDA
	0x0D22: "ddh",         // MALAYALAM LETTER DDHA
	0x0D23: "nn",          // MALAYALAM LETTER NNA
	0x0D24: "t",           // MALAYALAM LETTER TA
	0x0D25: "th",          // MALAYALAM LETTER THA
	0x0D26: "d",           // MALAYALAM LETTER DA
	0x0D27: "dh",          // MALAYALAM LETTER DHA
	0x0D28: "n",           // MALAYALAM LETTER NA
	0x0D2A: "p",           // MALAYALAM LETTER PA
	0x0D2B: "ph",          // MALAYALAM LETTER PHA
	0x0D2C: "b",           // MALAYALAM LETTER BA
	0x0D2D: "bh",          // MALAYALAM LETTER BHA
	0x0D2E: "m",           // MALAYALAM LETTER MA
	0x0D2F: "y",           // MALAYALAM LETTER YA
	0x0D30: "r",           // MALAYALAM LETTER RA
	0x0D31: "rr",          // MALAYALAM LETTER RRA
	0x0D32: "l",           // MALAYALAM LETTER LA
	0x0D33: "ll",          // MALAYALAM LETTER LLA
	0x0D34: "lll",         // MALAYALAM LETTER LLLA
	0x0D35: "v",           // MALAYALAM LETTER VA
	0x0D36: "sh",          // MALAYALAM LETTER SHA
	0x0D37: "ss",          // MALAYALAM LETTER SSA
	0x0D38: "s",           // MALAYALAM LETTER SA
	0x0D39: "h",           // MALAYALAM LETTER HA
	0x0D3E: "aa",          // MALAYALAM VOWEL SIGN AA
	0x0D3F: "i",           // MALAYALAM VOWEL SIGN I
	0x0D40: "ii",          // MALAYALAM VOWEL SIGN II
	0x0D41: "u",           // MALAYALAM VOWEL SIGN U
	0x0D42: "uu",          // MALAYALAM VOWEL SIGN UU
	0x0D43: "R",           // MALAYALAM VOWEL SIGN VOCALIC R
	0x0D46: "e",           // MALAYALAM VOWEL SIGN E
	0x0D47: "ee",          // MALAYALAM VOWEL SIGN EE
	0x0D48: "ai",          // MALAYALAM VOWEL SIGN AI
	0x0D4A: "o",           // MALAYALAM VOWEL SIGN O
	0x0D4B: "oo",          // MALAYALAM VOWEL SIGN OO
	0x0D4C: "au",          // MALAYALAM VOWEL SIGN AU
	0x0D57: "+",           // MALAYALAM AU LENGTH MARK
	0x0D60: "RR",          // MALAYALAM LETTER VOCALIC RR
	0x0D61: "LL",          // MALAYALAM LETTER VOCALIC LL
	0x0D66: "0",           // MALAYALAM DIGIT ZERO
	0x0D67: "1",           // MALAYALAM DIGIT ONE
	0x0D68: "2",           // MALAYALAM DIGIT TWO
	0x0D69: "3",           // MALAYALAM DIGIT THREE
	0x0D6A: "4",           // MALAYALAM DIGIT FOUR
	0x0D6B: "5",           // MALAYALAM DIGIT FIVE
	0x0D6C: "6",           // MALAYALAM DIGIT SIX
	0x0D6D: "7",           // MALAYALAM DIGIT SEVEN
	0x0D6E: "8",           // MALAYALAM DIGIT EIGHT
	0x0D6F: "9",           // MALAYALAM DIGIT NINE
	0x0D82: "N",           // SINHALA SIGN ANUSVARAYA
	0x0D83: "H",           // SINHALA SIGN VISARGAYA
	0x0D85: "a",           // SINHALA LETTER AYANNA
	0x0D86: "aa",          // SINHALA LETTER AAYANNA
	0x0D87: "ae",          // SINHALA LETTER AEYANNA
	0x0D88: "aae",         // SINHALA LETTER AEEYANNA
	0x0D89: "i",           // SINHALA LETTER IYANNA
	0x0D8A: "ii",          // SINHALA LETTER IIYANNA
	0x0D8B: "u",           // SINHALA LETTER UYANNA
	0x0D8C: "uu",          // SINHALA LETTER UUYANNA
	0x0D8D: "R",           // SINHALA LETTER IRUYANNA
	0x0D8E: "RR",          // SINHALA LETTER IRUUYANNA
	0x0D8F: "L",           // SINHALA LETTER ILUYANNA
	0x0D90: "LL",          // SINHALA LETTER ILUUYANNA
	0x0D91: "e",           // SINHALA LETTER EYANNA
	0x0D92: "ee",          // SINHALA LETTER EEYANNA
	0x0D93: "ai",          // SINHALA LETTER AIYANNA
	0x0D94: "o",           // SINHALA LETTER OYANNA
	0x0D95: "oo",          // SINHALA LETTER OOYANNA
	0x0D96: "au",          // SINHALA LETTER AUYANNA
	0x0D9A: "k",           // SINHALA LETTER ALPAPRAANA KAYANNA
	0x0D9B: "kh",          // SINHALA LETTER MAHAAPRAANA KAYANNA
	0x0D9C: "g",           // SINHALA LETTER ALPAPRAANA GAYANNA
	0x0D9D: "gh",          // SINHALA LETTER MAHAAPRAANA GAYANNA
	0x0D9E: "ng",          // SINHALA LETTER KANTAJA NAASIKYAYA
	0x0D9F: "nng",         // SINHALA LETTER SANYAKA GAYANNA
	0x0DA0: "c",           // SINHALA LETTER ALPAPRAANA CAYANNA
	0x0DA1: "ch",          // SINHALA LETTER MAHAAPRAANA CAYANNA
	0x0DA2: "j",           // SINHALA LETTER ALPAPRAANA JAYANNA
	0x0DA3: "jh",          // SINHALA LETTER MAHAAPRAANA JAYANNA
	0x0DA4: "ny",          // SINHALA LETTER TAALUJA NAASIKYAYA
	0x0DA5: "jny",         // SINHALA LETTER TAALUJA SANYOOGA NAAKSIKYAYA
	0x0DA6: "nyj",         // SINHALA LETTER SANYAKA JAYANNA
	0x0DA7: "tt",          // SINHALA LETTER ALPAPRAANA TTAYANNA
	0x0DA8: "tth",         // SINHALA LETTER MAHAAPRAANA TTAYANNA
	0x0DA9: "dd",          // SINHALA LETTER ALPAPRAANA DDAYANNA
	0x0DAA: "ddh",         // SINHALA LETTER MAHAAPRAANA DDAYANNA
	0x0DAB: "nn",          // SINHALA LETTER MUURDHAJA NAYANNA
	0x0DAC: "nndd",        // SINHALA LETTER SANYAKA DDAYANNA
	0x0DAD: "t",           // SINHALA LETTER ALPAPRAANA TAYANNA
	0x0DAE: "th",          // SINHALA LETTER MAHAAPRAANA TAYANNA
	0x0DAF: "d",           // SINHALA LETTER ALPAPRAANA DAYANNA
	0x0DB0: "dh",          // SINHALA LETTER MAHAAPRAANA DAYANNA
	0x0DB1: "n",           // SINHALA LETTER DANTAJA NAYANNA
	0x0DB3: "nd",          // SINHALA LETTER SANYAKA DAYANNA
	0x0DB4: "p",           // SINHALA LETTER ALPAPRAANA PAYANNA
	0x0DB5: "ph",          // SINHALA LETTER MAHAAPRAANA PAYANNA
	0x0DB6: "b",           // SINHALA LETTER ALPAPRAANA BAYANNA
	0x0DB7: "bh",          // SINHALA LETTER MAHAAPRAANA BAYANNA
	0x0DB8: "m",           // SINHALA LETTER MAYANNA
	0x0DB9: "mb",          // SINHALA LETTER AMBA BAYANNA
	0x0DBA: "y",           // SINHALA LETTER YAYANNA
	0x0DBB: "r",           // SINHALA LETTER RAYANNA
	0x0DBD: "l",           // SINHALA LETTER DANTAJA LAYANNA
	0x0DC0: "v",           // SINHALA LETTER VAYANNA
	0x0DC1: "sh",          // SINHALA LETTER TAALUJA SAYANNA
	0x0DC2: "ss",          // SINHALA LETTER MUURDHAJA SAYANNA
	0x0DC3: "s",           // SINHALA LETTER DANTAJA SAYANNA
	0x0DC4: "h",           // SINHALA LETTER HAYANNA
	0x0DC5: "ll",          // SINHALA LETTER MUURDHAJA LAYANNA
	0x0DC6: "f",           // SINHALA LETTER FAYANNA
	0x0DCF: "aa",          // SINHALA VOWEL SIGN AELA-PILLA
	0x0DD0: "ae",          // SINHALA VOWEL SIGN KETTI AEDA-PILLA
	0x0DD1: "aae",         // SINHALA VOWEL SIGN DIGA AEDA-PILLA
	0x0DD2: "i",           // SINHALA VOWEL SIGN KETTI IS-PILLA
	0x0DD3: "ii",          // SINHALA VOWEL SIGN DIGA IS-PILLA
	0x0DD4: "u",           // SINHALA VOWEL SIGN KETTI PAA-PILLA
	0x0DD6: "uu",          // SINHALA VOWEL SIGN DIGA PAA-PILLA
	0x0DD8: "R",           // SINHALA VOWEL SIGN GAETTA-PILLA
	0x0DD9: "e",           // SINHALA VOWEL SIGN KOMBUVA
	0x0DDA: "ee",          // SINHALA VOWEL SIGN DIGA KOMBUVA
	0x0DDB: "ai",          // SINHALA VOWEL SIGN KOMBU DEKA
	0x0DDC: "o",           // SINHALA VOWEL SIGN KOMBUVA HAA AELA-PILLA
	0x0DDD: "oo",          // SINHALA VOWEL SIGN KOMBUVA HAA DIGA AELA-PILLA
	0x0DDE: "au",          // SINHALA VOWEL SIGN KOMBUVA HAA GAYANUKITTA
	0x0DDF: "L",           // SINHALA VOWEL SIGN GAYANUKITTA
	0x0DF2: "RR",          // SINHALA VOWEL SIGN DIGA GAETTA-PILLA
	0x0DF3: "LL",          // SINHALA VOWEL SIGN DIGA GAYANUKITTA
	0x0DF4: " . ",         // SINHALA PUNCTUATION KUNDDALIYA
	0x0E01: "k",           // THAI CHARACTER KO KAI
	0x0E02: "kh",          // THAI CHARACTER KHO KHAI
	0x0E03: "kh",          // THAI CHARACTER KHO KHUAT
	0x0E04: "kh",          // THAI CHARACTER KHO KHWAI
	0x0E05: "kh",          // THAI CHARACTER KHO KHON
	0x0E06: "kh",          // THAI CHARACTER KHO RAKHANG
	0x0E07: "ng",          // THAI CHARACTER NGO NGU
	0x0E08: "cch",         // THAI CHARACTER CHO CHAN
	0x0E09: "ch",          // THAI CHARACTER CHO CHING
	0x0E0A: "ch",          // THAI CHARACTER CHO CHANG
	0x0E0B: "ch",          // THAI CHARACTER SO SO
	0x0E0C: "ch",          // THAI CHARACTER CHO CHOE
	0x0E0D: "y",           // THAI CHARACTER YO YING
	0x0E0E: "d",           // THAI CHARACTER DO CHADA
	0x0E0F: "t",           // THAI CHARACTER TO PATAK
	0x0E10: "th",          // THAI CHARACTER THO THAN
	0x0E11: "th",          // THAI CHARACTER THO NANGMONTHO
	0x0E12: "th",          // THAI CHARACTER THO PHUTHAO
	0x0E13: "n",           // THAI CHARACTER NO NEN
	0x0E14: "d",           // THAI CHARACTER DO DEK
	0x0E15: "t",           // THAI CHARACTER TO TAO
	0x0E16: "th",          // THAI CHARACTER THO THUNG
	0x0E17: "th",          // THAI CHARACTER THO THAHAN
	0x0E18: "th",          // THAI CHARACTER THO THONG
	0x0E19: "n",           // THAI CHARACTER NO NU
	0x0E1A: "b",           // THAI CHARACTER BO BAIMAI
	0x0E1B: "p",           // THAI CHARACTER PO PLA
	0x0E1C: "ph",          // THAI CHARACTER PHO PHUNG
	0x0E1D: "f",           // THAI CHARACTER FO FA
	0x0E1E: "ph",          // THAI CHARACTER PHO PHAN
	0x0E1F: "f",           // THAI CHARACTER FO FAN
	0x0E20: "ph",          // THAI CHARACTER PHO SAMPHAO
	0x0E21: "m",           // THAI CHARACTER MO MA
	0x0E22: "y",           // THAI CHARACTER YO YAK
	0x0E23: "r",           // THAI CHARACTER RO RUA
	0x0E24: "R",           // THAI CHARACTER RU
	0x0E25: "l",           // THAI CHARACTER LO LING
	0x0E26: "L",           // THAI CHARACTER LU
	0x0E27: "w",           // THAI CHARACTER WO WAEN
	0x0E28: "s",           // THAI CHARACTER SO SALA
	0x0E29: "s",           // THAI CHARACTER SO RUSI
	0x0E2A: "s",           // THAI CHARACTER SO SUA
	0x0E2B: "h",           // THAI CHARACTER HO HIP
	0x0E2C: "l",           // THAI CHARACTER LO CHULA
	0x0E2D: "`",           // THAI CHARACTER O ANG
	0x0E2E: "h",           // THAI CHARACTER HO NOKHUK
	0x0E2F: "~",           // THAI CHARACTER PAIYANNOI
	0x0E30: "a",           // THAI CHARACTER SARA A
	0x0E31: "a",           // THAI CHARACTER MAI HAN-AKAT
	0x0E32: "aa",          // THAI CHARACTER SARA AA
	0x0E33: "am",          // THAI CHARACTER SARA AM
	0x0E34: "i",           // THAI CHARACTER SARA I
	0x0E35: "ii",          // THAI CHARACTER SARA II
	0x0E36: "ue",          // THAI CHARACTER SARA UE
	0x0E37: "uue",         // THAI CHARACTER SARA UEE
	0x0E38: "u",           // THAI CHARACTER SARA U
	0x0E39: "uu",          // THAI CHARACTER SARA UU
	0x0E3F: "Bh.",         // THAI CURRENCY SYMBOL BAHT
	0x0E40: "e",           // THAI CHARACTER SARA E
	0x0E41: "ae",          // THAI CHARACTER SARA AE
	0x0E42: "o",           // THAI CHARACTER SARA O
	0x0E43: "ai",          // THAI CHARACTER SARA AI MAIMUAN
	0x0E44: "ai",          // THAI CHARACTER SARA AI MAIMALAI
	0x0E45: "ao",          // THAI CHARACTER LAKKHANGYAO
	0x0E46: "+",           // THAI CHARACTER MAIYAMOK
	0x0E4D: "M",           // THAI CHARACTER NIKHAHIT
	0x0E4F: " * ",         // THAI CHARACTER FONGMAN
	0x0E50: "0",           // THAI DIGIT ZERO
	0x0E51: "1",           // THAI DIGIT ONE
	0x0E52: "2",           // THAI DIGIT TWO
	0x0E53: "3",           // THAI DIGIT THREE
	0x0E54: "4",           // THAI DIGIT FOUR
	0x0E55: "5",           // THAI DIGIT FIVE
	0x0E56: "6",           // THAI DIGIT SIX
	0x0E57: "7",           // THAI DIGIT SEVEN
	0x0E58: "8",           // THAI DIGIT EIGHT
	0x0E59: "9",           // THAI DIGIT NINE
	0x0E5A: " // ",        // THAI CHARACTER ANGKHANKHU
	0x0E5B: " /// ",       // THAI CHARACTER KHOMUT
	0x0E81: "k",           // LAO LETTER KO
	0x0E82: "kh",          // LAO LETTER KHO SUNG
	0x0E84: "kh",          // LAO LETTER KHO TAM
	0x0E87: "ng",          // LAO LETTER NGO
	0x0E88: "ch",          // LAO LETTER CO
	0x0E8A: "s",           // LAO LETTER SO TAM
	0x0E8D: "ny",          // LAO LETTER NYO
	0x0E94: "d",           // LAO LETTER DO
	0x0E95: "h",           // LAO LETTER TO
	0x0E96: "th",          // LAO LETTER THO SUNG
	0x0E97: "th",          // LAO LETTER THO TAM
	0x0E99: "n",           // LAO LETTER NO
	0x0E9A: "b",           // LAO LETTER BO
	0x0E9B: "p",           // LAO LETTER PO
	0x0E9C: "ph",          // LAO LETTER PHO SUNG
	0x0E9D: "f",           // LAO LETTER FO TAM
	0x0E9E: "ph",          // LAO LETTER PHO TAM
	0x0E9F: "f",           // LAO LETTER FO SUNG
	0x0EA1: "m",           // LAO LETTER MO
	0x0EA2: "y",           // LAO LETTER YO
	0x0EA3: "r",           // LAO LETTER LO LING
	0x0EA5: "l",           // LAO LETTER LO LOOT
	0x0EA7: "w",           // LAO LETTER WO
	0x0EAA: "s",           // LAO LETTER SO SUNG
	0x0EAB: "h",           // LAO LETTER HO SUNG
	0x0EAD: "`",           // LAO LETTER O
	0x0EAF: "~",           // LAO ELLIPSIS
	0x0EB0: "a",           // LAO VOWEL SIGN A
	0x0EB2: "aa",          // LAO VOWEL SIGN AA
	0x0EB3: "am",          // LAO VOWEL SIGN AM
	0x0EB4: "i",           // LAO VOWEL SIGN I
	0x0EB5: "ii",          // LAO VOWEL SIGN II
	0x0EB6: "y",           // LAO VOWEL SIGN Y
	0x0EB7: "yy",          // LAO VOWEL SIGN YY
	0x0EB8: "u",           // LAO VOWEL SIGN U
	0x0EB9: "uu",          // LAO VOWEL SIGN UU
	0x0EBB: "o",           // LAO VOWEL SIGN MAI KON
	0x0EBC: "l",           // LAO SEMIVOWEL SIGN LO
	0x0EBD: "ny",          // LAO SEMIVOWEL SIGN NYO
	0x0EC0: "e",           // LAO VOWEL SIGN E
	0x0EC1: "ei",          // LAO VOWEL SIGN EI
	0x0EC2: "o",           // LAO VOWEL SIGN O
	0x0EC3: "ay",          // LAO VOWEL SIGN AY
	0x0EC4: "ai",          // LAO VOWEL SIGN AI
	0x0EC6: "+",           // LAO KO LA
	0x0ECD: "M",           // LAO NIGGAHITA
	0x0ED0: "0",           // LAO DIGIT ZERO
	0x0ED1: "1",           // LAO DIGIT ONE
	0x0ED2: "2",           // LAO DIGIT TWO
	0x0ED3: "3",           // LAO DIGIT THREE
	0x0ED4: "4",           // LAO DIGIT FOUR
	0x0ED5: "5",           // LAO DIGIT FIVE
	0x0ED6: "6",           // LAO DIGIT SIX
	0x0ED7: "7",           // LAO DIGIT SEVEN
	0x0ED8: "8",           // LAO DIGIT EIGHT
	0x0ED9: "9",           // LAO DIGIT NINE
	0x0EDC: "hn",          // LAO HO NO
	0x0EDD: "hm",          // LAO HO MO
	0x0F00: "AUM",         // TIBETAN SYLLABLE OM
	0x0F08: " // ",        // TIBETAN MARK SBRUL SHAD
	0x0F09: " * ",         // TIBETAN MARK BSKUR YIG MGO
	0x0F0B: "-",           // TIBETAN MARK INTERSYLLABIC TSHEG
	0x0F0C: " / ",         // TIBETAN MARK DELIMITER TSHEG BSTAR
	0x0F0D: " / ",         // TIBETAN MARK SHAD
	0x0F0E: " // ",        // TIBETAN MARK NYIS SHAD
	0x0F0F: " -/ ",        // TIBETAN MARK TSHEG SHAD
	0x0F10: " +/ ",        // TIBETAN MARK NYIS TSHEG SHAD
	0x0F11: " X/ ",        // TIBETAN MARK RIN CHEN SPUNGS SHAD
	0x0F12: " /XX/ ",      // TIBETAN MARK RGYA GRAM SHAD
	0x0F13: " /X/ ",       // TIBETAN MARK CARET -DZUD RTAGS ME LONG CAN
	0x0F14: ", ",          // TIBETAN MARK GTER TSHEG
	0x0F20: "0",           // TIBETAN DIGIT ZERO
	0x0F21: "1",           // TIBETAN DIGIT ONE
	0x0F22: "2",           // TIBETAN DIGIT TWO
	0x0F23: "3",           // TIBETAN DIGIT THREE
	0x0F24: "4",           // TIBETAN DIGIT FOUR
	0x0F25: "5",           // TIBETAN DIGIT FIVE
	0x0F26: "6",           // TIBETAN DIGIT SIX
	0x0F27: "7",           // TIBETAN DIGIT SEVEN
	0x0F28: "8",           // TIBETAN DIGIT EIGHT
	0x0F29: "9",           // TIBETAN DIGIT NINE
	0x0F2A: ".5",          // TIBETAN DIGIT HALF ONE
	0x0F2B: "1.5",         // TIBETAN DIGIT HALF TWO
	0x0F2C: "2.5",         // TIBETAN DIGIT HALF THREE
	0x0F2D: "3.5",         // TIBETAN DIGIT HALF FOUR
	0x0F2E: "4.5",         // TIBETAN DIGIT HALF FIVE
	0x0F2F: "5.5",         // TIBETAN DIGIT HALF SIX
	0x0F30: "6.5",         // TIBETAN DIGIT HALF SEVEN
	0x0F31: "7.5",         // TIBETAN DIGIT HALF EIGHT
	0x0F32: "8.5",         // TIBETAN DIGIT HALF NINE
	0x0F33: "-.5",         // TIBETAN DIGIT HALF ZERO
	0x0F34: "+",           // TIBETAN MARK BSDUS RTAGS
	0x0F35: "*",           // TIBETAN MARK NGAS BZUNG NYI ZLA
	0x0F36: "^",           // TIBETAN MARK CARET -DZUD RTAGS BZHI MIG CAN
	0x0F37: "_",           // TIBETAN MARK NGAS BZUNG SGOR RTAGS
	0x0F39: "~",           // TIBETAN MARK TSA -PHRU
	0x0F3B: "]",           // TIBETAN MARK GUG RTAGS GYAS
	0x0F3C: "[[",          // TIBETAN MARK ANG KHANG GYON
	0x0F3D: "]]",          // TIBETAN MARK ANG KHANG GYAS
	0x0F40: "k",           // TIBETAN LETTER KA
	0x0F41: "kh",          // TIBETAN LETTER KHA
	0x0F42: "g",           // TIBETAN LETTER GA
	0x0F43: "gh",          // TIBETAN LETTER GHA
	0x0F44: "ng",          // TIBETAN LETTER NGA
	0x0F45: "c",           // TIBETAN LETTER CA
	0x0F46: "ch",          // TIBETAN LETTER CHA
	0x0F47: "j",           // TIBETAN LETTER JA
	0x0F49: "ny",          // TIBETAN LETTER NYA
	0x0F4A: "tt",          // TIBETAN LETTER TTA
	0x0F4B: "tth",         // TIBETAN LETTER TTHA
	0x0F4C: "dd",          // TIBETAN LETTER DDA
	0x0F4D: "ddh",         // TIBETAN LETTER DDHA
	0x0F4E: "nn",          // TIBETAN LETTER NNA
	0x0F4F: "t",           // TIBETAN LETTER TA
	0x0F50: "th",          // TIBETAN LETTER THA
	0x0F51: "d",           // TIBETAN LETTER DA
	0x0F52: "dh",          // TIBETAN LETTER DHA
	0x0F53: "n",           // TIBETAN LETTER NA
	0x0F54: "p",           // TIBETAN LETTER PA
	0x0F55: "ph",          // TIBETAN LETTER PHA
	0x0F56: "b",           // TIBETAN LETTER BA
	0x0F57: "bh",          // TIBETAN LETTER BHA
	0x0F58: "m",           // TIBETAN LETTER MA
	0x0F59: "ts",          // TIBETAN LETTER TSA
	0x0F5A: "tsh",         // TIBETAN LETTER TSHA
	0x0F5B: "dz",          // TIBETAN LETTER DZA
	0x0F5C: "dzh",         // TIBETAN LETTER DZHA
	0x0F5D: "w",           // TIBETAN LETTER WA
	0x0F5E: "zh",          // TIBETAN LETTER ZHA
	0x0F5F: "z",           // TIBETAN LETTER ZA
	0x0F60: "'",           // TIBETAN LETTER -A
	0x0F61: "y",           // TIBETAN LETTER YA
	0x0F62: "r",           // TIBETAN LETTER RA
	0x0F63: "l",           // TIBETAN LETTER LA
	0x0F64: "sh",          // TIBETAN LETTER SHA
	0x0F65: "ssh",         // TIBETAN LETTER SSA
	0x0F66: "s",           // TIBETAN LETTER SA
	0x0F67: "h",           // TIBETAN LETTER HA
	0x0F68: "a",           // TIBETAN LETTER A
	0x0F69: "kss",         // TIBETAN LETTER KSSA
	0x0F6A: "r",           // TIBETAN LETTER FIXED-FORM RA
	0x0F71: "aa",          // TIBETAN VOWEL SIGN AA
	0x0F72: "i",           // TIBETAN VOWEL SIGN I
	0x0F73: "ii",          // TIBETAN VOWEL SIGN II
	0x0F74: "u",           // TIBETAN VOWEL SIGN U
	0x0F75: "uu",          // TIBETAN VOWEL SIGN UU
	0x0F76: "R",           // TIBETAN VOWEL SIGN VOCALIC R
	0x0F77: "RR",          // TIBETAN VOWEL SIGN VOCALIC RR
	0x0F78: "L",           // TIBETAN VOWEL SIGN VOCALIC L
	0x0F79: "LL",          // TIBETAN VOWEL SIGN VOCALIC LL
	0x0F7A: "e",           // TIBETAN VOWEL SIGN E
	0x0F7B: "ee",          // TIBETAN VOWEL SIGN EE
	0x0F7C: "o",           // TIBETAN VOWEL SIGN O
	0x0F7D: "oo",          // TIBETAN VOWEL SIGN OO
	0x0F7E: "M",           // TIBETAN SIGN RJES SU NGA RO
	0x0F7F: "H",           // TIBETAN SIGN RNAM BCAD
	0x0F80: "i",           // TIBETAN VOWEL SIGN REVERSED I
	0x0F81: "ii",          // TIBETAN VOWEL SIGN REVERSED II
	0x0F90: "k",           // TIBETAN SUBJOINED LETTER KA
	0x0F91: "kh",          // TIBETAN SUBJOINED LETTER KHA
	0x0F92: "g",           // TIBETAN SUBJOINED LETTER GA
	0x0F93: "gh",          // TIBETAN SUBJOINED LETTER GHA
	0x0F94: "ng",          // TIBETAN SUBJOINED LETTER NGA
	0x0F95: "c",           // TIBETAN SUBJOINED LETTER CA
	0x0F96: "ch",          // TIBETAN SUBJOINED LETTER CHA
	0x0F97: "j",           // TIBETAN SUBJOINED LETTER JA
	0x0F99: "ny",          // TIBETAN SUBJOINED LETTER NYA
	0x0F9A: "tt",          // TIBETAN SUBJOINED LETTER TTA
	0x0F9B: "tth",         // TIBETAN SUBJOINED LETTER TTHA
	0x0F9C: "dd",          // TIBETAN SUBJOINED LETTER DDA
	0x0F9D: "ddh",         // TIBETAN SUBJOINED LETTER DDHA
	0x0F9E: "nn",          // TIBETAN SUBJOINED LETTER NNA
	0x0F9F: "t",           // TIBETAN SUBJOINED LETTER TA
	0x0FA0: "th",          // TIBETAN SUBJOINED LETTER THA
	0x0FA1: "d",           // TIBETAN SUBJOINED LETTER DA
	0x0FA2: "dh",          // TIBETAN SUBJOINED LETTER DHA
	0x0FA3: "n",           // TIBETAN SUBJOINED LETTER NA
	0x0FA4: "p",           // TIBETAN SUBJOINED LETTER PA
	0x0FA5: "ph",          // TIBETAN SUBJOINED LETTER PHA
	0x0FA6: "b",           // TIBETAN SUBJOINED LETTER BA
	0x0FA7: "bh",          // TIBETAN SUBJOINED LETTER BHA
	0x0FA8: "m",           // TIBETAN SUBJOINED LETTER MA
	0x0FA9: "ts",          // TIBETAN SUBJOINED LETTER TSA
	0x0FAA: "tsh",         // TIBETAN SUBJOINED LETTER TSHA
	0x0FAB: "dz",          // TIBETAN SUBJOINED LETTER DZA
	0x0FAC: "dzh",         // TIBETAN SUBJOINED LETTER DZHA
	0x0FAD: "w",           // TIBETAN SUBJOINED LETTER WA
	0x0FAE: "zh",          // TIBETAN SUBJOINED LETTER ZHA
	0x0FAF: "z",           // TIBETAN SUBJOINED LETTER ZA
	0x0FB0: "'",           // TIBETAN SUBJOINED LETTER -A
	0x0FB1: "y",           // TIBETAN SUBJOINED LETTER YA
	0x0FB2: "r",           // TIBETAN SUBJOINED LETTER RA
	0x0FB3: "l",           // TIBETAN SUBJOINED LETTER LA
	0x0FB4: "sh",          // TIBETAN SUBJOINED LETTER SHA
	0x0FB5: "ss",          // TIBETAN SUBJOINED LETTER SSA
	0x0FB6: "s",           // TIBETAN SUBJOINED LETTER SA
	0x0FB7: "h",           // TIBETAN SUBJOINED LETTER HA
	0x0FB8: "a",           // TIBETAN SUBJOINED LETTER A
	0x0FB9: "kss",         // TIBETAN SUBJOINED LETTER KSSA
	0x0FBA: "w",           // TIBETAN SUBJOINED LETTER FIXED-FORM WA
	0x0FBB: "y",           // TIBETAN SUBJOINED LETTER FIXED-FORM YA
	0x0FBC: "r",           // TIBETAN SUBJOINED LETTER FIXED-FORM RA
	0x0FBE: "X",           // TIBETAN KU RU KHA
	0x0FBF: " :X: ",       // TIBETAN KU RU KHA BZHI MIG CAN
	0x0FC0: " /O/ ",       // TIBETAN CANTILLATION SIGN HEAVY BEAT
	0x0FC1: " /o/ ",       // TIBETAN CANTILLATION SIGN LIGHT BEAT
	0x0FC2: " \\o\\ ",     // TIBETAN CANTILLATION SIGN CANG TE-U
	0x0FC3: " (O) ",       // TIBETAN CANTILLATION SIGN SBUB -CHAL
	0x1000: "k",           // MYANMAR LETTER KA
	0x1001: "kh",          // MYANMAR LETTER KHA
	0x1002: "g",           // MYANMAR LETTER GA
	0x1003: "gh",          // MYANMAR LETTER GHA
	0x1004: "ng",          // MYANMAR LETTER NGA
	0x1005: "c",           // MYANMAR LETTER CA
	0x1006: "ch",          // MYANMAR LETTER CHA
	0x1007: "j",           // MYANMAR LETTER JA
	0x1008: "jh",          // MYANMAR LETTER JHA
	0x1009: "ny",          // MYANMAR LETTER NYA
	0x100A: "nny",         // MYANMAR LETTER NNYA
	0x100B: "tt",          // MYANMAR LETTER TTA
	0x100C: "tth",         // MYANMAR LETTER TTHA
	0x100D: "dd",          // MYANMAR LETTER DDA
	0x100E: "ddh",         // MYANMAR LETTER DDHA
	0x100F: "nn",          // MYANMAR LETTER NNA
	0x1010: "tt",          // MYANMAR LETTER TA
	0x1011: "th",          // MYANMAR LETTER THA
	0x1012: "d",           // MYANMAR LETTER DA
	0x1013: "dh",          // MYANMAR LETTER DHA
	0x1014: "n",           // MYANMAR LETTER NA
	0x1015: "p",           // MYANMAR LETTER PA
	0x1016: "ph",          // MYANMAR LETTER PHA
	0x1017: "b",           // MYANMAR LETTER BA
	0x1018: "bh",          // MYANMAR LETTER BHA
	0x1019: "m",           // MYANMAR LETTER MA
	0x101A: "y",           // MYANMAR LETTER YA
	0x101B: "r",           // MYANMAR LETTER RA
	0x101C: "l",           // MYANMAR LETTER LA
	0x101D: "w",           // MYANMAR LETTER WA
	0x101E: "s",           // MYANMAR LETTER SA
	0x101F: "h",           // MYANMAR LETTER HA
	0x1020: "ll",          // MYANMAR LETTER LLA
	0x1021: "a",           // MYANMAR LETTER A
	0x1023: "i",           // MYANMAR LETTER I
	0x1024: "ii",          // MYANMAR LETTER II
	0x1025: "u",           // MYANMAR LETTER U
	0x1026: "uu",          // MYANMAR LETTER UU
	0x1027: "e",           // MYANMAR LETTER E
	0x1029: "o",           // MYANMAR LETTER O
	0x102A: "au",          // MYANMAR LETTER AU
	0x102C: "aa",          // MYANMAR VOWEL SIGN AA
	0x102D: "i",           // MYANMAR VOWEL SIGN I
	0x102E: "ii",          // MYANMAR VOWEL SIGN II
	0x102F: "u",           // MYANMAR VOWEL SIGN U
	0x1030: "uu",          // MYANMAR VOWEL SIGN UU
	0x1031: "e",           // MYANMAR VOWEL SIGN E
	0x1032: "ai",          // MYANMAR VOWEL SIGN AI
	0x1036: "N",           // MYANMAR SIGN ANUSVARA
	0x1037: "'",           // MYANMAR SIGN DOT BELOW
	0x1038: ":",           // MYANMAR SIGN VISARGA
	0x1040: "0",           // MYANMAR DIGIT ZERO
	0x1041: "1",           // MYANMAR DIGIT ONE
	0x1042: "2",           // MYANMAR DIGIT TWO
	0x1043: "3",           // MYANMAR DIGIT THREE
	0x1044: "4",           // MYANMAR DIGIT FOUR
	0x1045: "5",           // MYANMAR DIGIT FIVE
	0x1046: "6",           // MYANMAR DIGIT SIX
	0x1047: "7",           // MYANMAR DIGIT SEVEN
	0x1048: "8",           // MYANMAR DIGIT EIGHT
	0x1049: "9",           // MYANMAR DIGIT NINE
	0x104A: " / ",         // MYANMAR SIGN LITTLE SECTION
	0x104B: " // ",        // MYANMAR SIGN SECTION
	0x104C: "n*",          // MYANMAR SYMBOL LOCATIVE
	0x104D: "r*",          // MYANMAR SYMBOL COMPLETED
	0x104E: "l*",          // MYANMAR SYMBOL AFOREMENTIONED
	0x104F: "e*",          // MYANMAR SYMBOL GENITIVE
	0x1050: "sh",          // MYANMAR LETTER SHA
	0x1051: "ss",          // MYANMAR LETTER SSA
	0x1052: "R",           // MYANMAR LETTER VOCALIC R
	0x1053: "RR",          // MYANMAR LETTER VOCALIC RR
	0x1054: "L",           // MYANMAR LETTER VOCALIC L
	0x1055: "LL",          // MYANMAR LETTER VOCALIC LL
	0x1056: "R",           // MYANMAR VOWEL SIGN VOCALIC R
	0x1057: "RR",          // MYANMAR VOWEL SIGN VOCALIC RR
	0x1058: "L",           // MYANMAR VOWEL SIGN VOCALIC L
	0x1059: "LL",          // MYANMAR VOWEL SIGN VOCALIC LL
	0x10A0: "A",           // GEORGIAN CAPITAL LETTER AN
	0x10A1: "B",           // GEORGIAN CAPITAL LETTER BAN
	0x10A2: "G",           // GEORGIAN CAPITAL LETTER GAN
	0x10A3: "D",           // GEORGIAN CAPITAL LETTER DON
	0x10A4: "E",           // GEORGIAN CAPITAL LETTER EN
	0x10A5: "V",           // GEORGIAN CAPITAL LETTER VIN
	0x10A6: "Z",           // GEORGIAN CAPITAL LETTER ZEN
	0x10A7: "T`",          // GEORGIAN CAPITAL LETTER TAN
	0x10A8: "I",           // GEORGIAN CAPITAL LETTER IN
	0x10A9: "K",           // GEORGIAN CAPITAL LETTER KAN
	0x10AA: "L",           // GEORGIAN CAPITAL LETTER LAS
	0x10AB: "M",           // GEORGIAN CAPITAL LETTER MAN
	0x10AC: "N",           // GEORGIAN CAPITAL LETTER NAR
	0x10AD: "O",           // GEORGIAN CAPITAL LETTER ON
	0x10AE: "P",           // GEORGIAN CAPITAL LETTER PAR
	0x10AF: "Zh",          // GEORGIAN CAPITAL LETTER ZHAR
	0x10B0: "R",           // GEORGIAN CAPITAL LETTER RAE
	0x10B1: "S",           // GEORGIAN CAPITAL LETTER SAN
	0x10B2: "T",           // GEORGIAN CAPITAL LETTER TAR
	0x10B3: "U",           // GEORGIAN CAPITAL LETTER UN
	0x10B4: "P`",          // GEORGIAN CAPITAL LETTER PHAR
	0x10B5: "K`",          // GEORGIAN CAPITAL LETTER KHAR
	0x10B6: "G'",          // GEORGIAN CAPITAL LETTER GHAN
	0x10B7: "Q",           // GEORGIAN CAPITAL LETTER QAR
	0x10B8: "Sh",          // GEORGIAN CAPITAL LETTER SHIN
	0x10B9: "Ch`",         // GEORGIAN CAPITAL LETTER CHIN
	0x10BA: "C`",          // GEORGIAN CAPITAL LETTER CAN
	0x10BB: "Z'",          // GEORGIAN CAPITAL LETTER JIL
	0x10BC: "C",           // GEORGIAN CAPITAL LETTER CIL
	0x10BD: "Ch",          // GEORGIAN CAPITAL LETTER CHAR
	0x10BE: "X",           // GEORGIAN CAPITAL LETTER XAN
	0x10BF: "J",           // GEORGIAN CAPITAL LETTER JHAN
	0x10C0: "H",           // GEORGIAN CAPITAL LETTER HAE
	0x10C1: "E",           // GEORGIAN CAPITAL LETTER HE
	0x10C2: "Y",           // GEORGIAN CAPITAL LETTER HIE
	0x10C3: "W",           // GEORGIAN CAPITAL LETTER WE
	0x10C4: "Xh",          // GEORGIAN CAPITAL LETTER HAR
	0x10C5: "OE",          // GEORGIAN CAPITAL LETTER HOE
	0x10D0: "a",           // GEORGIAN LETTER AN
	0x10D1: "b",           // GEORGIAN LETTER BAN
	0x10D2: "g",           // GEORGIAN LETTER GAN
	0x10D3: "d",           // GEORGIAN LETTER DON
	0x10D4: "e",           // GEORGIAN LETTER EN
	0x10D5: "v",           // GEORGIAN LETTER VIN
	0x10D6: "z",           // GEORGIAN LETTER ZEN
	0x10D7: "t`",          // GEORGIAN LETTER TAN
	0x10D8: "i",           // GEORGIAN LETTER IN
	0x10D9: "k",           // GEORGIAN LETTER KAN
	0x10DA: "l",           // GEORGIAN LETTER LAS
	0x10DB: "m",           // GEORGIAN LETTER MAN
	0x10DC: "n",           // GEORGIAN LETTER NAR
	0x10DD: "o",           // GEORGIAN LETTER ON
	0x10DE: "p",           // GEORGIAN LETTER PAR
	0x10DF: "zh",          // GEORGIAN LETTER ZHAR
	0x10E0: "r",           // GEORGIAN LETTER RAE
	0x10E1: "s",           // GEORGIAN LETTER SAN
	0x10E2: "t",           // GEORGIAN LETTER TAR
	0x10E3: "u",           // GEORGIAN LETTER UN
	0x10E4: "p`",          // GEORGIAN LETTER PHAR
	0x10E5: "k`",          // GEORGIAN LETTER KHAR
	0x10E6: "g'",          // GEORGIAN LETTER GHAN
	0x10E7: "q",           // GEORGIAN LETTER QAR
	0x10E8: "sh",          // GEORGIAN LETTER SHIN
	0x10E9: "ch`",         // GEORGIAN LETTER CHIN
	0x10EA: "c`",          // GEORGIAN LETTER CAN
	0x10EB: "z'",          // GEORGIAN LETTER JIL
	0x10EC: "c",           // GEORGIAN LETTER CIL
	0x10ED: "ch",          // GEORGIAN LETTER CHAR
	0x10EE: "x",           // GEORGIAN LETTER XAN
	0x10EF: "j",           // GEORGIAN LETTER JHAN
	0x10F0: "h",           // GEORGIAN LETTER HAE
	0x10F1: "e",           // GEORGIAN LETTER HE
	0x10F2: "y",           // GEORGIAN LETTER HIE
	0x10F3: "w",           // GEORGIAN LETTER WE
	0x10F4: "xh",          // GEORGIAN LETTER HAR
	0x10F5: "oe",          // GEORGIAN LETTER HOE
	0x10F6: "f",           // GEORGIAN LETTER FI
	0x10FB: " // ",        // GEORGIAN PARAGRAPH SEPARATOR
	0x1100: "g",           // HANGUL CHOSEONG KIYEOK
	0x1101: "gg",          // HANGUL CHOSEONG SSANGKIYEOK
	0x1102: "n",           // HANGUL CHOSEONG NIEUN
	0x1103: "d",           // HANGUL CHOSEONG TIKEUT
	0x1104: "dd",          // HANGUL CHOSEONG SSANGTIKEUT
	0x1105: "r",           // HANGUL CHOSEONG RIEUL
	0x1106: "m",           // HANGUL CHOSEONG MIEUM
	0x1107: "b",           // HANGUL CHOSEONG PIEUP
	0x1108: "bb",          // HANGUL CHOSEONG SSANGPIEUP
	0x1109: "s",           // HANGUL CHOSEONG SIOS
	0x110A: "ss",          // HANGUL CHOSEONG SSANGSIOS
	0x110C: "j",           // HANGUL CHOSEONG CIEUC
	0x110D: "jj",          // HANGUL CHOSEONG SSANGCIEUC
	0x110E: "c",           // HANGUL CHOSEONG CHIEUCH
	0x110F: "k",           // HANGUL CHOSEONG KHIEUKH
	0x1110: "t",           // HANGUL CHOSEONG THIEUTH
	0x1111: "p",           // HANGUL CHOSEONG PHIEUPH
	0x1112: "h",           // HANGUL CHOSEONG HIEUH
	0x1113: "ng",          // HANGUL CHOSEONG NIEUN-KIYEOK
	0x1114: "nn",          // HANGUL CHOSEONG SSANGNIEUN
	0x1115: "nd",          // HANGUL CHOSEONG NIEUN-TIKEUT
	0x1116: "nb",          // HANGUL CHOSEONG NIEUN-PIEUP
	0x1117: "dg",          // HANGUL CHOSEONG TIKEUT-KIYEOK
	0x1118: "rn",          // HANGUL CHOSEONG RIEUL-NIEUN
	0x1119: "rr",          // HANGUL CHOSEONG SSANGRIEUL
	0x111A: "rh",          // HANGUL CHOSEONG RIEUL-HIEUH
	0x111B: "rN",          // HANGUL CHOSEONG KAPYEOUNRIEUL
	0x111C: "mb",          // HANGUL CHOSEONG MIEUM-PIEUP
	0x111D: "mN",          // HANGUL CHOSEONG KAPYEOUNMIEUM
	0x111E: "bg",          // HANGUL CHOSEONG PIEUP-KIYEOK
	0x111F: "bn",          // HANGUL CHOSEONG PIEUP-NIEUN
	0x1121: "bs",          // HANGUL CHOSEONG PIEUP-SIOS
	0x1122: "bsg",         // HANGUL CHOSEONG PIEUP-SIOS-KIYEOK
	0x1123: "bst",         // HANGUL CHOSEONG PIEUP-SIOS-TIKEUT
	0x1124: "bsb",         // HANGUL CHOSEONG PIEUP-SIOS-PIEUP
	0x1125: "bss",         // HANGUL CHOSEONG PIEUP-SSANGSIOS
	0x1126: "bsj",         // HANGUL CHOSEONG PIEUP-SIOS-CIEUC
	0x1127: "bj",          // HANGUL CHOSEONG PIEUP-CIEUC
	0x1128: "bc",          // HANGUL CHOSEONG PIEUP-CHIEUCH
	0x1129: "bt",          // HANGUL CHOSEONG PIEUP-THIEUTH
	0x112A: "bp",          // HANGUL CHOSEONG PIEUP-PHIEUPH
	0x112B: "bN",          // HANGUL CHOSEONG KAPYEOUNPIEUP
	0x112C: "bbN",         // HANGUL CHOSEONG KAPYEOUNSSANGPIEUP
	0x112D: "sg",          // HANGUL CHOSEONG SIOS-KIYEOK
	0x112E: "sn",          // HANGUL CHOSEONG SIOS-NIEUN
	0x112F: "sd",          // HANGUL CHOSEONG SIOS-TIKEUT
	0x1130: "sr",          // HANGUL CHOSEONG SIOS-RIEUL
	0x1131: "sm",          // HANGUL CHOSEONG SIOS-MIEUM
	0x1132: "sb",          // HANGUL CHOSEONG SIOS-PIEUP
	0x1133: "sbg",         // HANGUL CHOSEONG SIOS-PIEUP-KIYEOK
	0x1134: "sss",         // HANGUL CHOSEONG SIOS-SSANGSIOS
	0x1135: "s",           // HANGUL CHOSEONG SIOS-IEUNG
	0x1136: "sj",          // HANGUL CHOSEONG SIOS-CIEUC
	0x1137: "sc",          // HANGUL CHOSEONG SIOS-CHIEUCH
	0x1138: "sk",          // HANGUL CHOSEONG SIOS-KHIEUKH
	0x1139: "st",          // HANGUL CHOSEONG SIOS-THIEUTH
	0x113A: "sp",          // HANGUL CHOSEONG SIOS-PHIEUPH
	0x113B: "sh",          // HANGUL CHOSEONG SIOS-HIEUH
	0x1140: "Z",           // HANGUL CHOSEONG PANSIOS
	0x1141: "g",           // HANGUL CHOSEONG IEUNG-KIYEOK
	0x1142: "d",           // HANGUL CHOSEONG IEUNG-TIKEUT
	0x1143: "m",           // HANGUL CHOSEONG IEUNG-MIEUM
	0x1144: "b",           // HANGUL CHOSEONG IEUNG-PIEUP
	0x1145: "s",           // HANGUL CHOSEONG IEUNG-SIOS
	0x1146: "Z",           // HANGUL CHOSEONG IEUNG-PANSIOS
	0x1148: "j",           // HANGUL CHOSEONG IEUNG-CIEUC
	0x1149: "c",           // HANGUL CHOSEONG IEUNG-CHIEUCH
	0x114A: "t",           // HANGUL CHOSEONG IEUNG-THIEUTH
	0x114B: "p",           // HANGUL CHOSEONG IEUNG-PHIEUPH
	0x114C: "N",           // HANGUL CHOSEONG YESIEUNG
	0x114D: "j",           // HANGUL CHOSEONG CIEUC-IEUNG
	0x1152: "ck",          // HANGUL CHOSEONG CHIEUCH-KHIEUKH
	0x1153: "ch",          // HANGUL CHOSEONG CHIEUCH-HIEUH
	0x1156: "pb",          // HANGUL CHOSEONG PHIEUPH-PIEUP
	0x1157: "pN",          // HANGUL CHOSEONG KAPYEOUNPHIEUPH
	0x1158: "hh",          // HANGUL CHOSEONG SSANGHIEUH
	0x1159: "Q",           // HANGUL CHOSEONG YEORINHIEUH
	0x1161: "a",           // HANGUL JUNGSEONG A
	0x1162: "ae",          // HANGUL JUNGSEONG AE
	0x1163: "ya",          // HANGUL JUNGSEONG YA
	0x1164: "yae",         // HANGUL JUNGSEONG YAE
	0x1165: "eo",          // HANGUL JUNGSEONG EO
	0x1166: "e",           // HANGUL JUNGSEONG E
	0x1167: "yeo",         // HANGUL JUNGSEONG YEO
	0x1168: "ye",          // HANGUL JUNGSEONG YE
	0x1169: "o",           // HANGUL JUNGSEONG O
	0x116A: "wa",          // HANGUL JUNGSEONG WA
	0x116B: "wae",         // HANGUL JUNGSEONG WAE
	0x116C: "oe",          // HANGUL JUNGSEONG OE
	0x116D: "yo",          // HANGUL JUNGSEONG YO
	0x116E: "u",           // HANGUL JUNGSEONG U
	0x116F: "weo",         // HANGUL JUNGSEONG WEO
	0x1170: "we",          // HANGUL JUNGSEONG WE
	0x1171: "wi",          // HANGUL JUNGSEONG WI
	0x1172: "yu",          // HANGUL JUNGSEONG YU
	0x1173: "eu",          // HANGUL JUNGSEONG EU
	0x1174: "yi",          // HANGUL JUNGSEONG YI
	0x1175: "i",           // HANGUL JUNGSEONG I
	0x1176: "a-o",         // HANGUL JUNGSEONG A-O
	0x1177: "a-u",         // HANGUL JUNGSEONG A-U
	0x1178: "ya-o",        // HANGUL JUNGSEONG YA-O
	0x1179: "ya-yo",       // HANGUL JUNGSEONG YA-YO
	0x117A: "eo-o",        // HANGUL JUNGSEONG EO-O
	0x117B: "eo-u",        // HANGUL JUNGSEONG EO-U
	0x117C: "eo-eu",       // HANGUL JUNGSEONG EO-EU
	0x117D: "yeo-o",       // HANGUL JUNGSEONG YEO-O
	0x117E: "yeo-u",       // HANGUL JUNGSEONG YEO-U
	0x117F: "o-eo",        // HANGUL JUNGSEONG O-EO
	0x1180: "o-e",         // HANGUL JUNGSEONG O-E
	0x1181: "o-ye",        // HANGUL JUNGSEONG O-YE
	0x1182: "o-o",         // HANGUL JUNGSEONG O-O
	0x1183: "o-u",         // HANGUL JUNGSEONG O-U
	0x1184: "yo-ya",       // HANGUL JUNGSEONG YO-YA
	0x1185: "yo-yae",      // HANGUL JUNGSEONG YO-YAE
	0x1186: "yo-yeo",      // HANGUL JUNGSEONG YO-YEO
	0x1187: "yo-o",        // HANGUL JUNGSEONG YO-O
	0x1188: "yo-i",        // HANGUL JUNGSEONG YO-I
	0x1189: "u-a",         // HANGUL JUNGSEONG U-A
	0x118A: "u-ae",        // HANGUL JUNGSEONG U-AE
	0x118B: "u-eo-eu",     // HANGUL JUNGSEONG U-EO-EU
	0x118C: "u-ye",        // HANGUL JUNGSEONG U-YE
	0x118D: "u-u",         // HANGUL JUNGSEONG U-U
	0x118E: "yu-a",        // HANGUL JUNGSEONG YU-A
	0x118F: "yu-eo",       // HANGUL JUNGSEONG YU-EO
	0x1190: "yu-e",        // HANGUL JUNGSEONG YU-E
	0x1191: "yu-yeo",      // HANGUL JUNGSEONG YU-YEO
	0x1192: "yu-ye",       // HANGUL JUNGSEONG YU-YE
	0x1193: "yu-u",        // HANGUL JUNGSEONG YU-U
	0x1194: "yu-i",        // HANGUL JUNGSEONG YU-I
	0x1195: "eu-u",        // HANGUL JUNGSEONG EU-U
	0x1196: "eu-eu",       // HANGUL JUNGSEONG EU-EU
	0x1197: "yi-u",        // HANGUL JUNGSEONG YI-U
	0x1198: "i-a",         // HANGUL JUNGSEONG I-A
	0x1199: "i-ya",        // HANGUL JUNGSEONG I-YA
	0x119A: "i-o",         // HANGUL JUNGSEONG I-O
	0x119B: "i-u",         // HANGUL JUNGSEONG I-U
	0x119C: "i-eu",        // HANGUL JUNGSEONG I-EU
	0x119D: "i-U",         // HANGUL JUNGSEONG I-ARAEA
	0x119E: "U",           // HANGUL JUNGSEONG ARAEA
	0x119F: "U-eo",        // HANGUL JUNGSEONG ARAEA-EO
	0x11A0: "U-u",         // HANGUL JUNGSEONG ARAEA-U
	0x11A1: "U-i",         // HANGUL JUNGSEONG ARAEA-I
	0x11A2: "UU",          // HANGUL JUNGSEONG SSANGARAEA
	0x11A8: "g",           // HANGUL JONGSEONG KIYEOK
	0x11A9: "gg",          // HANGUL JONGSEONG SSANGKIYEOK
	0x11AA: "gs",          // HANGUL JONGSEONG KIYEOK-SIOS
	0x11AB: "n",           // HANGUL JONGSEONG NIEUN
	0x11AC: "nj",          // HANGUL JONGSEONG NIEUN-CIEUC
	0x11AD: "nh",          // HANGUL JONGSEONG NIEUN-HIEUH
	0x11AE: "d",           // HANGUL JONGSEONG TIKEUT
	0x11AF: "l",           // HANGUL JONGSEONG RIEUL
	0x11B0: "lg",          // HANGUL JONGSEONG RIEUL-KIYEOK
	0x11B1: "lm",          // HANGUL JONGSEONG RIEUL-MIEUM
	0x11B2: "lb",          // HANGUL JONGSEONG RIEUL-PIEUP
	0x11B3: "ls",          // HANGUL JONGSEONG RIEUL-SIOS
	0x11B4: "lt",          // HANGUL JONGSEONG RIEUL-THIEUTH
	0x11B5: "lp",          // HANGUL JONGSEONG RIEUL-PHIEUPH
	0x11B6: "lh",          // HANGUL JONGSEONG RIEUL-HIEUH
	0x11B7: "m",           // HANGUL JONGSEONG MIEUM
	0x11B8: "b",           // HANGUL JONGSEONG PIEUP
	0x11B9: "bs",          // HANGUL JONGSEONG PIEUP-SIOS
	0x11BA: "s",           // HANGUL JONGSEONG SIOS
	0x11BB: "ss",          // HANGUL JONGSEONG SSANGSIOS
	0x11BC: "ng",          // HANGUL JONGSEONG IEUNG
	0x11BD: "j",           // HANGUL JONGSEONG CIEUC
	0x11BE: "c",           // HANGUL JONGSEONG CHIEUCH
	0x11BF: "k",           // HANGUL JONGSEONG KHIEUKH
	0x11C0: "t",           // HANGUL JONGSEONG THIEUTH
	0x11C1: "p",           // HANGUL JONGSEONG PHIEUPH
	0x11C2: "h",           // HANGUL JONGSEONG HIEUH
	0x11C3: "gl",          // HANGUL JONGSEONG KIYEOK-RIEUL
	0x11C4: "gsg",         // HANGUL JONGSEONG KIYEOK-SIOS-KIYEOK
	0x11C5: "ng",          // HANGUL JONGSEONG NIEUN-KIYEOK
	0x11C6: "nd",          // HANGUL JONGSEONG NIEUN-TIKEUT
	0x11C7: "ns",          // HANGUL JONGSEONG NIEUN-SIOS
	0x11C8: "nZ",          // HANGUL JONGSEONG NIEUN-PANSIOS
	0x11C9: "nt",          // HANGUL JONGSEONG NIEUN-THIEUTH
	0x11CA: "dg",          // HANGUL JONGSEONG TIKEUT-KIYEOK
	0x11CB: "tl",          // HANGUL JONGSEONG TIKEUT-RIEUL
	0x11CC: "lgs",         // HANGUL JONGSEONG RIEUL-KIYEOK-SIOS
	0x11CD: "ln",          // HANGUL JONGSEONG RIEUL-NIEUN
	0x11CE: "ld",          // HANGUL JONGSEONG RIEUL-TIKEUT
	0x11CF: "lth",         // HANGUL JONGSEONG RIEUL-TIKEUT-HIEUH
	0x11D0: "ll",          // HANGUL JONGSEONG SSANGRIEUL
	0x11D1: "lmg",         // HANGUL JONGSEONG RIEUL-MIEUM-KIYEOK
	0x11D2: "lms",         // HANGUL JONGSEONG RIEUL-MIEUM-SIOS
	0x11D3: "lbs",         // HANGUL JONGSEONG RIEUL-PIEUP-SIOS
	0x11D4: "lbh",         // HANGUL JONGSEONG RIEUL-PIEUP-HIEUH
	0x11D5: "rNp",         // HANGUL JONGSEONG RIEUL-KAPYEOUNPIEUP
	0x11D6: "lss",         // HANGUL JONGSEONG RIEUL-SSANGSIOS
	0x11D7: "lZ",          // HANGUL JONGSEONG RIEUL-PANSIOS
	0x11D8: "lk",          // HANGUL JONGSEONG RIEUL-KHIEUKH
	0x11D9: "lQ",          // HANGUL JONGSEONG RIEUL-YEORINHIEUH
	0x11DA: "mg",          // HANGUL JONGSEONG MIEUM-KIYEOK
	0x11DB: "ml",          // HANGUL JONGSEONG MIEUM-RIEUL
	0x11DC: "mb",          // HANGUL JONGSEONG MIEUM-PIEUP
	0x11DD: "ms",          // HANGUL JONGSEONG MIEUM-SIOS
	0x11DE: "mss",         // HANGUL JONGSEONG MIEUM-SSANGSIOS
	0x11DF: "mZ",          // HANGUL JONGSEONG MIEUM-PANSIOS
	0x11E0: "mc",          // HANGUL JONGSEONG MIEUM-CHIEUCH
	0x11E1: "mh",          // HANGUL JONGSEONG MIEUM-HIEUH
	0x11E2: "mN",          // HANGUL JONGSEONG KAPYEOUNMIEUM
	0x11E3: "bl",          // HANGUL JONGSEONG PIEUP-RIEUL
	0x11E4: "bp",          // HANGUL JONGSEONG PIEUP-PHIEUPH
	0x11E5: "ph",          // HANGUL JONGSEONG PIEUP-HIEUH
	0x11E6: "pN",          // HANGUL JONGSEONG KAPYEOUNPIEUP
	0x11E7: "sg",          // HANGUL JONGSEONG SIOS-KIYEOK
	0x11E8: "sd",          // HANGUL JONGSEONG SIOS-TIKEUT
	0x11E9: "sl",          // HANGUL JONGSEONG SIOS-RIEUL
	0x11EA: "sb",          // HANGUL JONGSEONG SIOS-PIEUP
	0x11EB: "Z",           // HANGUL JONGSEONG PANSIOS
	0x11EC: "g",           // HANGUL JONGSEONG IEUNG-KIYEOK
	0x11ED: "ss",          // HANGUL JONGSEONG IEUNG-SSANGKIYEOK
	0x11EE: "",            // HANGUL JONGSEONG SSANGIEUNG
	0x11EF: "kh",          // HANGUL JONGSEONG IEUNG-KHIEUKH
	0x11F0: "N",           // HANGUL JONGSEONG YESIEUNG
	0x11F1: "Ns",          // HANGUL JONGSEONG YESIEUNG-SIOS
	0x11F2: "NZ",          // HANGUL JONGSEONG YESIEUNG-PANSIOS
	0x11F3: "pb",          // HANGUL JONGSEONG PHIEUPH-PIEUP
	0x11F4: "pN",          // HANGUL JONGSEONG KAPYEOUNPHIEUPH
	0x11F5: "hn",          // HANGUL JONGSEONG HIEUH-NIEUN
	0x11F6: "hl",          // HANGUL JONGSEONG HIEUH-RIEUL
	0x11F7: "hm",          // HANGUL JONGSEONG HIEUH-MIEUM
	0x11F8: "hb",          // HANGUL JONGSEONG HIEUH-PIEUP
	0x11F9: "Q",           // HANGUL JONGSEONG YEORINHIEUH
	0x1200: "ha",          // ETHIOPIC SYLLABLE HA
	0x1201: "hu",          // ETHIOPIC SYLLABLE HU
	0x1202: "hi",          // ETHIOPIC SYLLABLE HI
	0x1203: "haa",         // ETHIOPIC SYLLABLE HAA
	0x1204: "hee",         // ETHIOPIC SYLLABLE HEE
	0x1205: "he",          // ETHIOPIC SYLLABLE HE
	0x1206: "ho",          // ETHIOPIC SYLLABLE HO
	0x1208: "la",          // ETHIOPIC SYLLABLE LA
	0x1209: "lu",          // ETHIOPIC SYLLABLE LU
	0x120A: "li",          // ETHIOPIC SYLLABLE LI
	0x120B: "laa",         // ETHIOPIC SYLLABLE LAA
	0x120C: "lee",         // ETHIOPIC SYLLABLE LEE
	0x120D: "le",          // ETHIOPIC SYLLABLE LE
	0x120E: "lo",          // ETHIOPIC SYLLABLE LO
	0x120F: "lwa",         // ETHIOPIC SYLLABLE LWA
	0x1210: "hha",         // ETHIOPIC SYLLABLE HHA
	0x1211: "hhu",         // ETHIOPIC SYLLABLE HHU
	0x1212: "hhi",         // ETHIOPIC SYLLABLE HHI
	0x1213: "hhaa",        // ETHIOPIC SYLLABLE HHAA
	0x1214: "hhee",        // ETHIOPIC SYLLABLE HHEE
	0x1215: "hhe",         // ETHIOPIC SYLLABLE HHE
	0x1216: "hho",         // ETHIOPIC SYLLABLE HHO
	0x1217: "hhwa",        // ETHIOPIC SYLLABLE HHWA
	0x1218: "ma",          // ETHIOPIC SYLLABLE MA
	0x1219: "mu",          // ETHIOPIC SYLLABLE MU
	0x121A: "mi",          // ETHIOPIC SYLLABLE MI
	0x121B: "maa",         // ETHIOPIC SYLLABLE MAA
	0x121C: "mee",         // ETHIOPIC SYLLABLE MEE
	0x121D: "me",          // ETHIOPIC SYLLABLE ME
	0x121E: "mo",          // ETHIOPIC SYLLABLE MO
	0x121F: "mwa",         // ETHIOPIC SYLLABLE MWA
	0x1220: "sza",         // ETHIOPIC SYLLABLE SZA
	0x1221: "szu",         // ETHIOPIC SYLLABLE SZU
	0x1222: "szi",         // ETHIOPIC SYLLABLE SZI
	0x1223: "szaa",        // ETHIOPIC SYLLABLE SZAA
	0x1224: "szee",        // ETHIOPIC SYLLABLE SZEE
	0x1225: "sze",         // ETHIOPIC SYLLABLE SZE
	0x1226: "szo",         // ETHIOPIC SYLLABLE SZO
	0x1227: "szwa",        // ETHIOPIC SYLLABLE SZWA
	0x1228: "ra",          // ETHIOPIC SYLLABLE RA
	0x1229: "ru",          // ETHIOPIC SYLLABLE RU
	0x122A: "ri",          // ETHIOPIC SYLLABLE RI
	0x122B: "raa",         // ETHIOPIC SYLLABLE RAA
	0x122C: "ree",         // ETHIOPIC SYLLABLE REE
	0x122D: "re",          // ETHIOPIC SYLLABLE RE
	0x122E: "ro",          // ETHIOPIC SYLLABLE RO
	0x122F: "rwa",         // ETHIOPIC SYLLABLE RWA
	0x1230: "sa",          // ETHIOPIC SYLLABLE SA
	0x1231: "su",          // ETHIOPIC SYLLABLE SU
	0x1232: "si",          // ETHIOPIC SYLLABLE SI
	0x1233: "saa",         // ETHIOPIC SYLLABLE SAA
	0x1234: "see",         // ETHIOPIC SYLLABLE SEE
	0x1235: "se",          // ETHIOPIC SYLLABLE SE
	0x1236: "so",          // ETHIOPIC SYLLABLE SO
	0x1237: "swa",         // ETHIOPIC SYLLABLE SWA
	0x1238: "sha",         // ETHIOPIC SYLLABLE SHA
	0x1239: "shu",         // ETHIOPIC SYLLABLE SHU
	0x123A: "shi",         // ETHIOPIC SYLLABLE SHI
	0x123B: "shaa",        // ETHIOPIC SYLLABLE SHAA
	0x123C: "shee",        // ETHIOPIC SYLLABLE SHEE
	0x123D: "she",         // ETHIOPIC SYLLABLE SHE
	0x123E: "sho",         // ETHIOPIC SYLLABLE SHO
	0x123F: "shwa",        // ETHIOPIC SYLLABLE SHWA
	0x1240: "qa",          // ETHIOPIC SYLLABLE QA
	0x1241: "qu",          // ETHIOPIC SYLLABLE QU
	0x1242: "qi",          // ETHIOPIC SYLLABLE QI
	0x1243: "qaa",         // ETHIOPIC SYLLABLE QAA
	0x1244: "qee",         // ETHIOPIC SYLLABLE QEE
	0x1245: "qe",          // ETHIOPIC SYLLABLE QE
	0x1246: "qo",          // ETHIOPIC SYLLABLE QO
	0x1247: "",            //
	0x1248: "qwa",         // ETHIOPIC SYLLABLE QWA
	0x124A: "qwi",         // ETHIOPIC SYLLABLE QWI
	0x124B: "qwaa",        // ETHIOPIC SYLLABLE QWAA
	0x124C: "qwee",        // ETHIOPIC SYLLABLE QWEE
	0x124D: "qwe",         // ETHIOPIC SYLLABLE QWE
	0x1250: "qha",         // ETHIOPIC SYLLABLE QHA
	0x1251: "qhu",         // ETHIOPIC SYLLABLE QHU
	0x1252: "qhi",         // ETHIOPIC SYLLABLE QHI
	0x1253: "qhaa",        // ETHIOPIC SYLLABLE QHAA
	0x1254: "qhee",        // ETHIOPIC SYLLABLE QHEE
	0x1255: "qhe",         // ETHIOPIC SYLLABLE QHE
	0x1256: "qho",         // ETHIOPIC SYLLABLE QHO
	0x1258: "qhwa",        // ETHIOPIC SYLLABLE QHWA
	0x125A: "qhwi",        // ETHIOPIC SYLLABLE QHWI
	0x125B: "qhwaa",       // ETHIOPIC SYLLABLE QHWAA
	0x125C: "qhwee",       // ETHIOPIC SYLLABLE QHWEE
	0x125D: "qhwe",        // ETHIOPIC SYLLABLE QHWE
	0x1260: "ba",          // ETHIOPIC SYLLABLE BA
	0x1261: "bu",          // ETHIOPIC SYLLABLE BU
	0x1262: "bi",          // ETHIOPIC SYLLABLE BI
	0x1263: "baa",         // ETHIOPIC SYLLABLE BAA
	0x1264: "bee",         // ETHIOPIC SYLLABLE BEE
	0x1265: "be",          // ETHIOPIC SYLLABLE BE
	0x1266: "bo",          // ETHIOPIC SYLLABLE BO
	0x1267: "bwa",         // ETHIOPIC SYLLABLE BWA
	0x1268: "va",          // ETHIOPIC SYLLABLE VA
	0x1269: "vu",          // ETHIOPIC SYLLABLE VU
	0x126A: "vi",          // ETHIOPIC SYLLABLE VI
	0x126B: "vaa",         // ETHIOPIC SYLLABLE VAA
	0x126C: "vee",         // ETHIOPIC SYLLABLE VEE
	0x126D: "ve",          // ETHIOPIC SYLLABLE VE
	0x126E: "vo",          // ETHIOPIC SYLLABLE VO
	0x126F: "vwa",         // ETHIOPIC SYLLABLE VWA
	0x1270: "ta",          // ETHIOPIC SYLLABLE TA
	0x1271: "tu",          // ETHIOPIC SYLLABLE TU
	0x1272: "ti",          // ETHIOPIC SYLLABLE TI
	0x1273: "taa",         // ETHIOPIC SYLLABLE TAA
	0x1274: "tee",         // ETHIOPIC SYLLABLE TEE
	0x1275: "te",          // ETHIOPIC SYLLABLE TE
	0x1276: "to",          // ETHIOPIC SYLLABLE TO
	0x1277: "twa",         // ETHIOPIC SYLLABLE TWA
	0x1278: "ca",          // ETHIOPIC SYLLABLE CA
	0x1279: "cu",          // ETHIOPIC SYLLABLE CU
	0x127A: "ci",          // ETHIOPIC SYLLABLE CI
	0x127B: "caa",         // ETHIOPIC SYLLABLE CAA
	0x127C: "cee",         // ETHIOPIC SYLLABLE CEE
	0x127D: "ce",          // ETHIOPIC SYLLABLE CE
	0x127E: "co",          // ETHIOPIC SYLLABLE CO
	0x127F: "cwa",         // ETHIOPIC SYLLABLE CWA
	0x1280: "xa",          // ETHIOPIC SYLLABLE XA
	0x1281: "xu",          // ETHIOPIC SYLLABLE XU
	0x1282: "xi",          // ETHIOPIC SYLLABLE XI
	0x1283: "xaa",         // ETHIOPIC SYLLABLE XAA
	0x1284: "xee",         // ETHIOPIC SYLLABLE XEE
	0x1285: "xe",          // ETHIOPIC SYLLABLE XE
	0x1286: "xo",          // ETHIOPIC SYLLABLE XO
	0x1288: "xwa",         // ETHIOPIC SYLLABLE XWA
	0x128A: "xwi",         // ETHIOPIC SYLLABLE XWI
	0x128B: "xwaa",        // ETHIOPIC SYLLABLE XWAA
	0x128C: "xwee",        // ETHIOPIC SYLLABLE XWEE
	0x128D: "xwe",         // ETHIOPIC SYLLABLE XWE
	0x1290: "na",          // ETHIOPIC SYLLABLE NA
	0x1291: "nu",          // ETHIOPIC SYLLABLE NU
	0x1292: "ni",          // ETHIOPIC SYLLABLE NI
	0x1293: "naa",         // ETHIOPIC SYLLABLE NAA
	0x1294: "nee",         // ETHIOPIC SYLLABLE NEE
	0x1295: "ne",          // ETHIOPIC SYLLABLE NE
	0x1296: "no",          // ETHIOPIC SYLLABLE NO
	0x1297: "nwa",         // ETHIOPIC SYLLABLE NWA
	0x1298: "nya",         // ETHIOPIC SYLLABLE NYA
	0x1299: "nyu",         // ETHIOPIC SYLLABLE NYU
	0x129A: "nyi",         // ETHIOPIC SYLLABLE NYI
	0x129B: "nyaa",        // ETHIOPIC SYLLABLE NYAA
	0x129C: "nyee",        // ETHIOPIC SYLLABLE NYEE
	0x129D: "nye",         // ETHIOPIC SYLLABLE NYE
	0x129E: "nyo",         // ETHIOPIC SYLLABLE NYO
	0x129F: "nywa",        // ETHIOPIC SYLLABLE NYWA
	0x12A0: "'a",          // ETHIOPIC SYLLABLE GLOTTAL A
	0x12A1: "'u",          // ETHIOPIC SYLLABLE GLOTTAL U
	0x12A3: "'aa",         // ETHIOPIC SYLLABLE GLOTTAL AA
	0x12A4: "'ee",         // ETHIOPIC SYLLABLE GLOTTAL EE
	0x12A5: "'e",          // ETHIOPIC SYLLABLE GLOTTAL E
	0x12A6: "'o",          // ETHIOPIC SYLLABLE GLOTTAL O
	0x12A7: "'wa",         // ETHIOPIC SYLLABLE GLOTTAL WA
	0x12A8: "ka",          // ETHIOPIC SYLLABLE KA
	0x12A9: "ku",          // ETHIOPIC SYLLABLE KU
	0x12AA: "ki",          // ETHIOPIC SYLLABLE KI
	0x12AB: "kaa",         // ETHIOPIC SYLLABLE KAA
	0x12AC: "kee",         // ETHIOPIC SYLLABLE KEE
	0x12AD: "ke",          // ETHIOPIC SYLLABLE KE
	0x12AE: "ko",          // ETHIOPIC SYLLABLE KO
	0x12B0: "kwa",         // ETHIOPIC SYLLABLE KWA
	0x12B2: "kwi",         // ETHIOPIC SYLLABLE KWI
	0x12B3: "kwaa",        // ETHIOPIC SYLLABLE KWAA
	0x12B4: "kwee",        // ETHIOPIC SYLLABLE KWEE
	0x12B5: "kwe",         // ETHIOPIC SYLLABLE KWE
	0x12B8: "kxa",         // ETHIOPIC SYLLABLE KXA
	0x12B9: "kxu",         // ETHIOPIC SYLLABLE KXU
	0x12BA: "kxi",         // ETHIOPIC SYLLABLE KXI
	0x12BB: "kxaa",        // ETHIOPIC SYLLABLE KXAA
	0x12BC: "kxee",        // ETHIOPIC SYLLABLE KXEE
	0x12BD: "kxe",         // ETHIOPIC SYLLABLE KXE
	0x12BE: "kxo",         // ETHIOPIC SYLLABLE KXO
	0x12C0: "kxwa",        // ETHIOPIC SYLLABLE KXWA
	0x12C2: "kxwi",        // ETHIOPIC SYLLABLE KXWI
	0x12C3: "kxwaa",       // ETHIOPIC SYLLABLE KXWAA
	0x12C4: "kxwee",       // ETHIOPIC SYLLABLE KXWEE
	0x12C5: "kxwe",        // ETHIOPIC SYLLABLE KXWE
	0x12C8: "wa",          // ETHIOPIC SYLLABLE WA
	0x12C9: "wu",          // ETHIOPIC SYLLABLE WU
	0x12CA: "wi",          // ETHIOPIC SYLLABLE WI
	0x12CB: "waa",         // ETHIOPIC SYLLABLE WAA
	0x12CC: "wee",         // ETHIOPIC SYLLABLE WEE
	0x12CD: "we",          // ETHIOPIC SYLLABLE WE
	0x12CE: "wo",          // ETHIOPIC SYLLABLE WO
	0x12D0: "`a",          // ETHIOPIC SYLLABLE PHARYNGEAL A
	0x12D1: "`u",          // ETHIOPIC SYLLABLE PHARYNGEAL U
	0x12D2: "`i",          // ETHIOPIC SYLLABLE PHARYNGEAL I
	0x12D3: "`aa",         // ETHIOPIC SYLLABLE PHARYNGEAL AA
	0x12D4: "`ee",         // ETHIOPIC SYLLABLE PHARYNGEAL EE
	0x12D5: "`e",          // ETHIOPIC SYLLABLE PHARYNGEAL E
	0x12D6: "`o",          // ETHIOPIC SYLLABLE PHARYNGEAL O
	0x12D8: "za",          // ETHIOPIC SYLLABLE ZA
	0x12D9: "zu",          // ETHIOPIC SYLLABLE ZU
	0x12DA: "zi",          // ETHIOPIC SYLLABLE ZI
	0x12DB: "zaa",         // ETHIOPIC SYLLABLE ZAA
	0x12DC: "zee",         // ETHIOPIC SYLLABLE ZEE
	0x12DD: "ze",          // ETHIOPIC SYLLABLE ZE
	0x12DE: "zo",          // ETHIOPIC SYLLABLE ZO
	0x12DF: "zwa",         // ETHIOPIC SYLLABLE ZWA
	0x12E0: "zha",         // ETHIOPIC SYLLABLE ZHA
	0x12E1: "zhu",         // ETHIOPIC SYLLABLE ZHU
	0x12E2: "zhi",         // ETHIOPIC SYLLABLE ZHI
	0x12E3: "zhaa",        // ETHIOPIC SYLLABLE ZHAA
	0x12E4: "zhee",        // ETHIOPIC SYLLABLE ZHEE
	0x12E5: "zhe",         // ETHIOPIC SYLLABLE ZHE
	0x12E6: "zho",         // ETHIOPIC SYLLABLE ZHO
	0x12E7: "zhwa",        // ETHIOPIC SYLLABLE ZHWA
	0x12E8: "ya",          // ETHIOPIC SYLLABLE YA
	0x12E9: "yu",          // ETHIOPIC SYLLABLE YU
	0x12EA: "yi",          // ETHIOPIC SYLLABLE YI
	0x12EB: "yaa",         // ETHIOPIC SYLLABLE YAA
	0x12EC: "yee",         // ETHIOPIC SYLLABLE YEE
	0x12ED: "ye",          // ETHIOPIC SYLLABLE YE
	0x12EE: "yo",          // ETHIOPIC SYLLABLE YO
	0x12F0: "da",          // ETHIOPIC SYLLABLE DA
	0x12F1: "du",          // ETHIOPIC SYLLABLE DU
	0x12F2: "di",          // ETHIOPIC SYLLABLE DI
	0x12F3: "daa",         // ETHIOPIC SYLLABLE DAA
	0x12F4: "dee",         // ETHIOPIC SYLLABLE DEE
	0x12F5: "de",          // ETHIOPIC SYLLABLE DE
	0x12F6: "do",          // ETHIOPIC SYLLABLE DO
	0x12F7: "dwa",         // ETHIOPIC SYLLABLE DWA
	0x12F8: "dda",         // ETHIOPIC SYLLABLE DDA
	0x12F9: "ddu",         // ETHIOPIC SYLLABLE DDU
	0x12FA: "ddi",         // ETHIOPIC SYLLABLE DDI
	0x12FB: "ddaa",        // ETHIOPIC SYLLABLE DDAA
	0x12FC: "ddee",        // ETHIOPIC SYLLABLE DDEE
	0x12FD: "dde",         // ETHIOPIC SYLLABLE DDE
	0x12FE: "ddo",         // ETHIOPIC SYLLABLE DDO
	0x12FF: "ddwa",        // ETHIOPIC SYLLABLE DDWA
	0x1300: "ja",          // ETHIOPIC SYLLABLE JA
	0x1301: "ju",          // ETHIOPIC SYLLABLE JU
	0x1302: "ji",          // ETHIOPIC SYLLABLE JI
	0x1303: "jaa",         // ETHIOPIC SYLLABLE JAA
	0x1304: "jee",         // ETHIOPIC SYLLABLE JEE
	0x1305: "je",          // ETHIOPIC SYLLABLE JE
	0x1306: "jo",          // ETHIOPIC SYLLABLE JO
	0x1307: "jwa",         // ETHIOPIC SYLLABLE JWA
	0x1308: "ga",          // ETHIOPIC SYLLABLE GA
	0x1309: "gu",          // ETHIOPIC SYLLABLE GU
	0x130A: "gi",          // ETHIOPIC SYLLABLE GI
	0x130B: "gaa",         // ETHIOPIC SYLLABLE GAA
	0x130C: "gee",         // ETHIOPIC SYLLABLE GEE
	0x130D: "ge",          // ETHIOPIC SYLLABLE GE
	0x130E: "go",          // ETHIOPIC SYLLABLE GO
	0x1310: "gwa",         // ETHIOPIC SYLLABLE GWA
	0x1312: "gwi",         // ETHIOPIC SYLLABLE GWI
	0x1313: "gwaa",        // ETHIOPIC SYLLABLE GWAA
	0x1314: "gwee",        // ETHIOPIC SYLLABLE GWEE
	0x1315: "gwe",         // ETHIOPIC SYLLABLE GWE
	0x1318: "gga",         // ETHIOPIC SYLLABLE GGA
	0x1319: "ggu",         // ETHIOPIC SYLLABLE GGU
	0x131A: "ggi",         // ETHIOPIC SYLLABLE GGI
	0x131B: "ggaa",        // ETHIOPIC SYLLABLE GGAA
	0x131C: "ggee",        // ETHIOPIC SYLLABLE GGEE
	0x131D: "gge",         // ETHIOPIC SYLLABLE GGE
	0x131E: "ggo",         // ETHIOPIC SYLLABLE GGO
	0x1320: "tha",         // ETHIOPIC SYLLABLE THA
	0x1321: "thu",         // ETHIOPIC SYLLABLE THU
	0x1322: "thi",         // ETHIOPIC SYLLABLE THI
	0x1323: "thaa",        // ETHIOPIC SYLLABLE THAA
	0x1324: "thee",        // ETHIOPIC SYLLABLE THEE
	0x1325: "the",         // ETHIOPIC SYLLABLE THE
	0x1326: "tho",         // ETHIOPIC SYLLABLE THO
	0x1327: "thwa",        // ETHIOPIC SYLLABLE THWA
	0x1328: "cha",         // ETHIOPIC SYLLABLE CHA
	0x1329: "chu",         // ETHIOPIC SYLLABLE CHU
	0x132A: "chi",         // ETHIOPIC SYLLABLE CHI
	0x132B: "chaa",        // ETHIOPIC SYLLABLE CHAA
	0x132C: "chee",        // ETHIOPIC SYLLABLE CHEE
	0x132D: "che",         // ETHIOPIC SYLLABLE CHE
	0x132E: "cho",         // ETHIOPIC SYLLABLE CHO
	0x132F: "chwa",        // ETHIOPIC SYLLABLE CHWA
	0x1330: "pha",         // ETHIOPIC SYLLABLE PHA
	0x1331: "phu",         // ETHIOPIC SYLLABLE PHU
	0x1332: "phi",         // ETHIOPIC SYLLABLE PHI
	0x1333: "phaa",        // ETHIOPIC SYLLABLE PHAA
	0x1334: "phee",        // ETHIOPIC SYLLABLE PHEE
	0x1335: "phe",         // ETHIOPIC SYLLABLE PHE
	0x1336: "pho",         // ETHIOPIC SYLLABLE PHO
	0x1337: "phwa",        // ETHIOPIC SYLLABLE PHWA
	0x1338: "tsa",         // ETHIOPIC SYLLABLE TSA
	0x1339: "tsu",         // ETHIOPIC SYLLABLE TSU
	0x133A: "tsi",         // ETHIOPIC SYLLABLE TSI
	0x133B: "tsaa",        // ETHIOPIC SYLLABLE TSAA
	0x133C: "tsee",        // ETHIOPIC SYLLABLE TSEE
	0x133D: "tse",         // ETHIOPIC SYLLABLE TSE
	0x133E: "tso",         // ETHIOPIC SYLLABLE TSO
	0x133F: "tswa",        // ETHIOPIC SYLLABLE TSWA
	0x1340: "tza",         // ETHIOPIC SYLLABLE TZA
	0x1341: "tzu",         // ETHIOPIC SYLLABLE TZU
	0x1342: "tzi",         // ETHIOPIC SYLLABLE TZI
	0x1343: "tzaa",        // ETHIOPIC SYLLABLE TZAA
	0x1344: "tzee",        // ETHIOPIC SYLLABLE TZEE
	0x1345: "tze",         // ETHIOPIC SYLLABLE TZE
	0x1346: "tzo",         // ETHIOPIC SYLLABLE TZO
	0x1348: "fa",          // ETHIOPIC SYLLABLE FA
	0x1349: "fu",          // ETHIOPIC SYLLABLE FU
	0x134A: "fi",          // ETHIOPIC SYLLABLE FI
	0x134B: "faa",         // ETHIOPIC SYLLABLE FAA
	0x134C: "fee",         // ETHIOPIC SYLLABLE FEE
	0x134D: "fe",          // ETHIOPIC SYLLABLE FE
	0x134E: "fo",          // ETHIOPIC SYLLABLE FO
	0x134F: "fwa",         // ETHIOPIC SYLLABLE FWA
	0x1350: "pa",          // ETHIOPIC SYLLABLE PA
	0x1351: "pu",          // ETHIOPIC SYLLABLE PU
	0x1352: "pi",          // ETHIOPIC SYLLABLE PI
	0x1353: "paa",         // ETHIOPIC SYLLABLE PAA
	0x1354: "pee",         // ETHIOPIC SYLLABLE PEE
	0x1355: "pe",          // ETHIOPIC SYLLABLE PE
	0x1356: "po",          // ETHIOPIC SYLLABLE PO
	0x1357: "pwa",         // ETHIOPIC SYLLABLE PWA
	0x1358: "rya",         // ETHIOPIC SYLLABLE RYA
	0x1359: "mya",         // ETHIOPIC SYLLABLE MYA
	0x135A: "fya",         // ETHIOPIC SYLLABLE FYA
	0x1362: ".",           // ETHIOPIC FULL STOP
	0x1363: ",",           // ETHIOPIC COMMA
	0x1364: ";",           // ETHIOPIC SEMICOLON
	0x1365: ":",           // ETHIOPIC COLON
	0x1366: ":: ",         // ETHIOPIC PREFACE COLON
	0x1367: "?",           // ETHIOPIC QUESTION MARK
	0x1368: "//",          // ETHIOPIC PARAGRAPH SEPARATOR
	0x1369: "1",           // ETHIOPIC DIGIT ONE
	0x136A: "2",           // ETHIOPIC DIGIT TWO
	0x136B: "3",           // ETHIOPIC DIGIT THREE
	0x136C: "4",           // ETHIOPIC DIGIT FOUR
	0x136D: "5",           // ETHIOPIC DIGIT FIVE
	0x136E: "6",           // ETHIOPIC DIGIT SIX
	0x136F: "7",           // ETHIOPIC DIGIT SEVEN
	0x1370: "8",           // ETHIOPIC DIGIT EIGHT
	0x1371: "9",           // ETHIOPIC DIGIT NINE
	0x1372: "10+",         // ETHIOPIC NUMBER TEN
	0x1373: "20+",         // ETHIOPIC NUMBER TWENTY
	0x1374: "30+",         // ETHIOPIC NUMBER THIRTY
	0x1375: "40+",         // ETHIOPIC NUMBER FORTY
	0x1376: "50+",         // ETHIOPIC NUMBER FIFTY
	0x1377: "60+",         // ETHIOPIC NUMBER SIXTY
	0x1378: "70+",         // ETHIOPIC NUMBER SEVENTY
	0x1379: "80+",         // ETHIOPIC NUMBER EIGHTY
	0x137A: "90+",         // ETHIOPIC NUMBER NINETY
	0x137B: "100+",        // ETHIOPIC NUMBER HUNDRED
	0x137C: "10,000+",     // ETHIOPIC NUMBER TEN THOUSAND
	0x13A0: "a",           // CHEROKEE LETTER A
	0x13A1: "e",           // CHEROKEE LETTER E
	0x13A2: "i",           // CHEROKEE LETTER I
	0x13A3: "o",           // CHEROKEE LETTER O
	0x13A4: "u",           // CHEROKEE LETTER U
	0x13A5: "v",           // CHEROKEE LETTER V
	0x13A6: "ga",          // CHEROKEE LETTER GA
	0x13A7: "ka",          // CHEROKEE LETTER KA
	0x13A8: "ge",          // CHEROKEE LETTER GE
	0x13A9: "gi",          // CHEROKEE LETTER GI
	0x13AA: "go",          // CHEROKEE LETTER GO
	0x13AB: "gu",          // CHEROKEE LETTER GU
	0x13AC: "gv",          // CHEROKEE LETTER GV
	0x13AD: "ha",          // CHEROKEE LETTER HA
	0x13AE: "he",          // CHEROKEE LETTER HE
	0x13AF: "hi",          // CHEROKEE LETTER HI
	0x13B0: "ho",          // CHEROKEE LETTER HO
	0x13B1: "hu",          // CHEROKEE LETTER HU
	0x13B2: "hv",          // CHEROKEE LETTER HV
	0x13B3: "la",          // CHEROKEE LETTER LA
	0x13B4: "le",          // CHEROKEE LETTER LE
	0x13B5: "li",          // CHEROKEE LETTER LI
	0x13B6: "lo",          // CHEROKEE LETTER LO
	0x13B7: "lu",          // CHEROKEE LETTER LU
	0x13B8: "lv",          // CHEROKEE LETTER LV
	0x13B9: "ma",          // CHEROKEE LETTER MA
	0x13BA: "me",          // CHEROKEE LETTER ME
	0x13BB: "mi",          // CHEROKEE LETTER MI
	0x13BC: "mo",          // CHEROKEE LETTER MO
	0x13BD: "mu",          // CHEROKEE LETTER MU
	0x13BE: "na",          // CHEROKEE LETTER NA
	0x13BF: "hna",         // CHEROKEE LETTER HNA
	0x13C0: "nah",         // CHEROKEE LETTER NAH
	0x13C1: "ne",          // CHEROKEE LETTER NE
	0x13C2: "ni",          // CHEROKEE LETTER NI
	0x13C3: "no",          // CHEROKEE LETTER NO
	0x13C4: "nu",          // CHEROKEE LETTER NU
	0x13C5: "nv",          // CHEROKEE LETTER NV
	0x13C6: "qua",         // CHEROKEE LETTER QUA
	0x13C7: "que",         // CHEROKEE LETTER QUE
	0x13C8: "qui",         // CHEROKEE LETTER QUI
	0x13C9: "quo",         // CHEROKEE LETTER QUO
	0x13CA: "quu",         // CHEROKEE LETTER QUU
	0x13CB: "quv",         // CHEROKEE LETTER QUV
	0x13CC: "sa",          // CHEROKEE LETTER SA
	0x13CD: "s",           // CHEROKEE LETTER S
	0x13CE: "se",          // CHEROKEE LETTER SE
	0x13CF: "si",          // CHEROKEE LETTER SI
	0x13D0: "so",          // CHEROKEE LETTER SO
	0x13D1: "su",          // CHEROKEE LETTER SU
	0x13D2: "sv",          // CHEROKEE LETTER SV
	0x13D3: "da",          // CHEROKEE LETTER DA
	0x13D4: "ta",          // CHEROKEE LETTER TA
	0x13D5: "de",          // CHEROKEE LETTER DE
	0x13D6: "te",          // CHEROKEE LETTER TE
	0x13D7: "di",          // CHEROKEE LETTER DI
	0x13D8: "ti",          // CHEROKEE LETTER TI
	0x13D9: "do",          // CHEROKEE LETTER DO
	0x13DA: "du",          // CHEROKEE LETTER DU
	0x13DB: "dv",          // CHEROKEE LETTER DV
	0x13DC: "dla",         // CHEROKEE LETTER DLA
	0x13DD: "tla",         // CHEROKEE LETTER TLA
	0x13DE: "tle",         // CHEROKEE LETTER TLE
	0x13DF: "tli",         // CHEROKEE LETTER TLI
	0x13E0: "tlo",         // CHEROKEE LETTER TLO
	0x13E1: "tlu",         // CHEROKEE LETTER TLU
	0x13E2: "tlv",         // CHEROKEE LETTER TLV
	0x13E3: "tsa",         // CHEROKEE LETTER TSA
	0x13E4: "tse",         // CHEROKEE LETTER TSE
	0x13E5: "tsi",         // CHEROKEE LETTER TSI
	0x13E6: "tso",         // CHEROKEE LETTER TSO
	0x13E7: "tsu",         // CHEROKEE LETTER TSU
	0x13E8: "tsv",         // CHEROKEE LETTER TSV
	0x13E9: "wa",          // CHEROKEE LETTER WA
	0x13EA: "we",          // CHEROKEE LETTER WE
	0x13EB: "wi",          // CHEROKEE LETTER WI
	0x13EC: "wo",          // CHEROKEE LETTER WO
	0x13ED: "wu",          // CHEROKEE LETTER WU
	0x13EE: "wv",          // CHEROKEE LETTER WV
	0x13EF: "ya",          // CHEROKEE LETTER YA
	0x13F0: "ye",          // CHEROKEE LETTER YE
	0x13F1: "yi",          // CHEROKEE LETTER YI
	0x13F2: "yo",          // CHEROKEE LETTER YO
	0x13F3: "yu",          // CHEROKEE LETTER YU
	0x13F4: "yv",          // CHEROKEE LETTER YV
	0x1401: "e",           // CANADIAN SYLLABICS E
	0x1402: "aai",         // CANADIAN SYLLABICS AAI
	0x1403: "i",           // CANADIAN SYLLABICS I
	0x1404: "ii",          // CANADIAN SYLLABICS II
	0x1405: "o",           // CANADIAN SYLLABICS O
	0x1406: "oo",          // CANADIAN SYLLABICS OO
	0x1407: "oo",          // CANADIAN SYLLABICS Y-CREE OO
	0x1408: "ee",          // CANADIAN SYLLABICS CARRIER EE
	0x1409: "i",           // CANADIAN SYLLABICS CARRIER I
	0x140A: "a",           // CANADIAN SYLLABICS A
	0x140B: "aa",          // CANADIAN SYLLABICS AA
	0x140C: "we",          // CANADIAN SYLLABICS WE
	0x140D: "we",          // CANADIAN SYLLABICS WEST-CREE WE
	0x140E: "wi",          // CANADIAN SYLLABICS WI
	0x140F: "wi",          // CANADIAN SYLLABICS WEST-CREE WI
	0x1410: "wii",         // CANADIAN SYLLABICS WII
	0x1411: "wii",         // CANADIAN SYLLABICS WEST-CREE WII
	0x1412: "wo",          // CANADIAN SYLLABICS WO
	0x1413: "wo",          // CANADIAN SYLLABICS WEST-CREE WO
	0x1414: "woo",         // CANADIAN SYLLABICS WOO
	0x1415: "woo",         // CANADIAN SYLLABICS WEST-CREE WOO
	0x1416: "woo",         // CANADIAN SYLLABICS NASKAPI WOO
	0x1417: "wa",          // CANADIAN SYLLABICS WA
	0x1418: "wa",          // CANADIAN SYLLABICS WEST-CREE WA
	0x1419: "waa",         // CANADIAN SYLLABICS WAA
	0x141A: "waa",         // CANADIAN SYLLABICS WEST-CREE WAA
	0x141B: "waa",         // CANADIAN SYLLABICS NASKAPI WAA
	0x141C: "ai",          // CANADIAN SYLLABICS AI
	0x141D: "w",           // CANADIAN SYLLABICS Y-CREE W
	0x141E: "'",           // CANADIAN SYLLABICS GLOTTAL STOP
	0x141F: "t",           // CANADIAN SYLLABICS FINAL ACUTE
	0x1420: "k",           // CANADIAN SYLLABICS FINAL GRAVE
	0x1421: "sh",          // CANADIAN SYLLABICS FINAL BOTTOM HALF RING
	0x1422: "s",           // CANADIAN SYLLABICS FINAL TOP HALF RING
	0x1423: "n",           // CANADIAN SYLLABICS FINAL RIGHT HALF RING
	0x1424: "w",           // CANADIAN SYLLABICS FINAL RING
	0x1425: "n",           // CANADIAN SYLLABICS FINAL DOUBLE ACUTE
	0x1427: "w",           // CANADIAN SYLLABICS FINAL MIDDLE DOT
	0x1428: "c",           // CANADIAN SYLLABICS FINAL SHORT HORIZONTAL STROKE
	0x1429: "?",           // CANADIAN SYLLABICS FINAL PLUS
	0x142A: "l",           // CANADIAN SYLLABICS FINAL DOWN TACK
	0x142B: "en",          // CANADIAN SYLLABICS EN
	0x142C: "in",          // CANADIAN SYLLABICS IN
	0x142D: "on",          // CANADIAN SYLLABICS ON
	0x142E: "an",          // CANADIAN SYLLABICS AN
	0x142F: "pe",          // CANADIAN SYLLABICS PE
	0x1430: "paai",        // CANADIAN SYLLABICS PAAI
	0x1431: "pi",          // CANADIAN SYLLABICS PI
	0x1432: "pii",         // CANADIAN SYLLABICS PII
	0x1433: "po",          // CANADIAN SYLLABICS PO
	0x1434: "poo",         // CANADIAN SYLLABICS POO
	0x1435: "poo",         // CANADIAN SYLLABICS Y-CREE POO
	0x1436: "hee",         // CANADIAN SYLLABICS CARRIER HEE
	0x1437: "hi",          // CANADIAN SYLLABICS CARRIER HI
	0x1438: "pa",          // CANADIAN SYLLABICS PA
	0x1439: "paa",         // CANADIAN SYLLABICS PAA
	0x143A: "pwe",         // CANADIAN SYLLABICS PWE
	0x143B: "pwe",         // CANADIAN SYLLABICS WEST-CREE PWE
	0x143C: "pwi",         // CANADIAN SYLLABICS PWI
	0x143D: "pwi",         // CANADIAN SYLLABICS WEST-CREE PWI
	0x143E: "pwii",        // CANADIAN SYLLABICS PWII
	0x143F: "pwii",        // CANADIAN SYLLABICS WEST-CREE PWII
	0x1440: "pwo",         // CANADIAN SYLLABICS PWO
	0x1441: "pwo",         // CANADIAN SYLLABICS WEST-CREE PWO
	0x1442: "pwoo",        // CANADIAN SYLLABICS PWOO
	0x1443: "pwoo",        // CANADIAN SYLLABICS WEST-CREE PWOO
	0x1444: "pwa",         // CANADIAN SYLLABICS PWA
	0x1445: "pwa",         // CANADIAN SYLLABICS WEST-CREE PWA
	0x1446: "pwaa",        // CANADIAN SYLLABICS PWAA
	0x1447: "pwaa",        // CANADIAN SYLLABICS WEST-CREE PWAA
	0x1448: "pwaa",        // CANADIAN SYLLABICS Y-CREE PWAA
	0x1449: "p",           // CANADIAN SYLLABICS P
	0x144A: "p",           // CANADIAN SYLLABICS WEST-CREE P
	0x144B: "h",           // CANADIAN SYLLABICS CARRIER H
	0x144C: "te",          // CANADIAN SYLLABICS TE
	0x144D: "taai",        // CANADIAN SYLLABICS TAAI
	0x144E: "ti",          // CANADIAN SYLLABICS TI
	0x144F: "tii",         // CANADIAN SYLLABICS TII
	0x1450: "to",          // CANADIAN SYLLABICS TO
	0x1451: "too",         // CANADIAN SYLLABICS TOO
	0x1452: "too",         // CANADIAN SYLLABICS Y-CREE TOO
	0x1453: "dee",         // CANADIAN SYLLABICS CARRIER DEE
	0x1454: "di",          // CANADIAN SYLLABICS CARRIER DI
	0x1455: "ta",          // CANADIAN SYLLABICS TA
	0x1456: "taa",         // CANADIAN SYLLABICS TAA
	0x1457: "twe",         // CANADIAN SYLLABICS TWE
	0x1458: "twe",         // CANADIAN SYLLABICS WEST-CREE TWE
	0x1459: "twi",         // CANADIAN SYLLABICS TWI
	0x145A: "twi",         // CANADIAN SYLLABICS WEST-CREE TWI
	0x145B: "twii",        // CANADIAN SYLLABICS TWII
	0x145C: "twii",        // CANADIAN SYLLABICS WEST-CREE TWII
	0x145D: "two",         // CANADIAN SYLLABICS TWO
	0x145E: "two",         // CANADIAN SYLLABICS WEST-CREE TWO
	0x145F: "twoo",        // CANADIAN SYLLABICS TWOO
	0x1460: "twoo",        // CANADIAN SYLLABICS WEST-CREE TWOO
	0x1461: "twa",         // CANADIAN SYLLABICS TWA
	0x1462: "twa",         // CANADIAN SYLLABICS WEST-CREE TWA
	0x1463: "twaa",        // CANADIAN SYLLABICS TWAA
	0x1464: "twaa",        // CANADIAN SYLLABICS WEST-CREE TWAA
	0x1465: "twaa",        // CANADIAN SYLLABICS NASKAPI TWAA
	0x1466: "t",           // CANADIAN SYLLABICS T
	0x1467: "tte",         // CANADIAN SYLLABICS TTE
	0x1468: "tti",         // CANADIAN SYLLABICS TTI
	0x1469: "tto",         // CANADIAN SYLLABICS TTO
	0x146A: "tta",         // CANADIAN SYLLABICS TTA
	0x146B: "ke",          // CANADIAN SYLLABICS KE
	0x146C: "kaai",        // CANADIAN SYLLABICS KAAI
	0x146D: "ki",          // CANADIAN SYLLABICS KI
	0x146E: "kii",         // CANADIAN SYLLABICS KII
	0x146F: "ko",          // CANADIAN SYLLABICS KO
	0x1470: "koo",         // CANADIAN SYLLABICS KOO
	0x1471: "koo",         // CANADIAN SYLLABICS Y-CREE KOO
	0x1472: "ka",          // CANADIAN SYLLABICS KA
	0x1473: "kaa",         // CANADIAN SYLLABICS KAA
	0x1474: "kwe",         // CANADIAN SYLLABICS KWE
	0x1475: "kwe",         // CANADIAN SYLLABICS WEST-CREE KWE
	0x1476: "kwi",         // CANADIAN SYLLABICS KWI
	0x1477: "kwi",         // CANADIAN SYLLABICS WEST-CREE KWI
	0x1478: "kwii",        // CANADIAN SYLLABICS KWII
	0x1479: "kwii",        // CANADIAN SYLLABICS WEST-CREE KWII
	0x147A: "kwo",         // CANADIAN SYLLABICS KWO
	0x147B: "kwo",         // CANADIAN SYLLABICS WEST-CREE KWO
	0x147C: "kwoo",        // CANADIAN SYLLABICS KWOO
	0x147D: "kwoo",        // CANADIAN SYLLABICS WEST-CREE KWOO
	0x147E: "kwa",         // CANADIAN SYLLABICS KWA
	0x147F: "kwa",         // CANADIAN SYLLABICS WEST-CREE KWA
	0x1480: "kwaa",        // CANADIAN SYLLABICS KWAA
	0x1481: "kwaa",        // CANADIAN SYLLABICS WEST-CREE KWAA
	0x1482: "kwaa",        // CANADIAN SYLLABICS NASKAPI KWAA
	0x1483: "k",           // CANADIAN SYLLABICS K
	0x1484: "kw",          // CANADIAN SYLLABICS KW
	0x1485: "keh",         // CANADIAN SYLLABICS SOUTH-SLAVEY KEH
	0x1486: "kih",         // CANADIAN SYLLABICS SOUTH-SLAVEY KIH
	0x1487: "koh",         // CANADIAN SYLLABICS SOUTH-SLAVEY KOH
	0x1488: "kah",         // CANADIAN SYLLABICS SOUTH-SLAVEY KAH
	0x1489: "ce",          // CANADIAN SYLLABICS CE
	0x148A: "caai",        // CANADIAN SYLLABICS CAAI
	0x148B: "ci",          // CANADIAN SYLLABICS CI
	0x148C: "cii",         // CANADIAN SYLLABICS CII
	0x148D: "co",          // CANADIAN SYLLABICS CO
	0x148E: "coo",         // CANADIAN SYLLABICS COO
	0x148F: "coo",         // CANADIAN SYLLABICS Y-CREE COO
	0x1490: "ca",          // CANADIAN SYLLABICS CA
	0x1491: "caa",         // CANADIAN SYLLABICS CAA
	0x1492: "cwe",         // CANADIAN SYLLABICS CWE
	0x1493: "cwe",         // CANADIAN SYLLABICS WEST-CREE CWE
	0x1494: "cwi",         // CANADIAN SYLLABICS CWI
	0x1495: "cwi",         // CANADIAN SYLLABICS WEST-CREE CWI
	0x1496: "cwii",        // CANADIAN SYLLABICS CWII
	0x1497: "cwii",        // CANADIAN SYLLABICS WEST-CREE CWII
	0x1498: "cwo",         // CANADIAN SYLLABICS CWO
	0x1499: "cwo",         // CANADIAN SYLLABICS WEST-CREE CWO
	0x149A: "cwoo",        // CANADIAN SYLLABICS CWOO
	0x149B: "cwoo",        // CANADIAN SYLLABICS WEST-CREE CWOO
	0x149C: "cwa",         // CANADIAN SYLLABICS CWA
	0x149D: "cwa",         // CANADIAN SYLLABICS WEST-CREE CWA
	0x149E: "cwaa",        // CANADIAN SYLLABICS CWAA
	0x149F: "cwaa",        // CANADIAN SYLLABICS WEST-CREE CWAA
	0x14A0: "cwaa",        // CANADIAN SYLLABICS NASKAPI CWAA
	0x14A1: "c",           // CANADIAN SYLLABICS C
	0x14A2: "th",          // CANADIAN SYLLABICS SAYISI TH
	0x14A3: "me",          // CANADIAN SYLLABICS ME
	0x14A4: "maai",        // CANADIAN SYLLABICS MAAI
	0x14A5: "mi",          // CANADIAN SYLLABICS MI
	0x14A6: "mii",         // CANADIAN SYLLABICS MII
	0x14A7: "mo",          // CANADIAN SYLLABICS MO
	0x14A8: "moo",         // CANADIAN SYLLABICS MOO
	0x14A9: "moo",         // CANADIAN SYLLABICS Y-CREE MOO
	0x14AA: "ma",          // CANADIAN SYLLABICS MA
	0x14AB: "maa",         // CANADIAN SYLLABICS MAA
	0x14AC: "mwe",         // CANADIAN SYLLABICS MWE
	0x14AD: "mwe",         // CANADIAN SYLLABICS WEST-CREE MWE
	0x14AE: "mwi",         // CANADIAN SYLLABICS MWI
	0x14AF: "mwi",         // CANADIAN SYLLABICS WEST-CREE MWI
	0x14B0: "mwii",        // CANADIAN SYLLABICS MWII
	0x14B1: "mwii",        // CANADIAN SYLLABICS WEST-CREE MWII
	0x14B2: "mwo",         // CANADIAN SYLLABICS MWO
	0x14B3: "mwo",         // CANADIAN SYLLABICS WEST-CREE MWO
	0x14B4: "mwoo",        // CANADIAN SYLLABICS MWOO
	0x14B5: "mwoo",        // CANADIAN SYLLABICS WEST-CREE MWOO
	0x14B6: "mwa",         // CANADIAN SYLLABICS MWA
	0x14B7: "mwa",         // CANADIAN SYLLABICS WEST-CREE MWA
	0x14B8: "mwaa",        // CANADIAN SYLLABICS MWAA
	0x14B9: "mwaa",        // CANADIAN SYLLABICS WEST-CREE MWAA
	0x14BA: "mwaa",        // CANADIAN SYLLABICS NASKAPI MWAA
	0x14BB: "m",           // CANADIAN SYLLABICS M
	0x14BC: "m",           // CANADIAN SYLLABICS WEST-CREE M
	0x14BD: "mh",          // CANADIAN SYLLABICS MH
	0x14BE: "m",           // CANADIAN SYLLABICS ATHAPASCAN M
	0x14BF: "m",           // CANADIAN SYLLABICS SAYISI M
	0x14C0: "ne",          // CANADIAN SYLLABICS NE
	0x14C1: "naai",        // CANADIAN SYLLABICS NAAI
	0x14C2: "ni",          // CANADIAN SYLLABICS NI
	0x14C3: "nii",         // CANADIAN SYLLABICS NII
	0x14C4: "no",          // CANADIAN SYLLABICS NO
	0x14C5: "noo",         // CANADIAN SYLLABICS NOO
	0x14C6: "noo",         // CANADIAN SYLLABICS Y-CREE NOO
	0x14C7: "na",          // CANADIAN SYLLABICS NA
	0x14C8: "naa",         // CANADIAN SYLLABICS NAA
	0x14C9: "nwe",         // CANADIAN SYLLABICS NWE
	0x14CA: "nwe",         // CANADIAN SYLLABICS WEST-CREE NWE
	0x14CB: "nwa",         // CANADIAN SYLLABICS NWA
	0x14CC: "nwa",         // CANADIAN SYLLABICS WEST-CREE NWA
	0x14CD: "nwaa",        // CANADIAN SYLLABICS NWAA
	0x14CE: "nwaa",        // CANADIAN SYLLABICS WEST-CREE NWAA
	0x14CF: "nwaa",        // CANADIAN SYLLABICS NASKAPI NWAA
	0x14D0: "n",           // CANADIAN SYLLABICS N
	0x14D1: "ng",          // CANADIAN SYLLABICS CARRIER NG
	0x14D2: "nh",          // CANADIAN SYLLABICS NH
	0x14D3: "le",          // CANADIAN SYLLABICS LE
	0x14D4: "laai",        // CANADIAN SYLLABICS LAAI
	0x14D5: "li",          // CANADIAN SYLLABICS LI
	0x14D6: "lii",         // CANADIAN SYLLABICS LII
	0x14D7: "lo",          // CANADIAN SYLLABICS LO
	0x14D8: "loo",         // CANADIAN SYLLABICS LOO
	0x14D9: "loo",         // CANADIAN SYLLABICS Y-CREE LOO
	0x14DA: "la",          // CANADIAN SYLLABICS LA
	0x14DB: "laa",         // CANADIAN SYLLABICS LAA
	0x14DC: "lwe",         // CANADIAN SYLLABICS LWE
	0x14DD: "lwe",         // CANADIAN SYLLABICS WEST-CREE LWE
	0x14DE: "lwi",         // CANADIAN SYLLABICS LWI
	0x14DF: "lwi",         // CANADIAN SYLLABICS WEST-CREE LWI
	0x14E0: "lwii",        // CANADIAN SYLLABICS LWII
	0x14E1: "lwii",        // CANADIAN SYLLABICS WEST-CREE LWII
	0x14E2: "lwo",         // CANADIAN SYLLABICS LWO
	0x14E3: "lwo",         // CANADIAN SYLLABICS WEST-CREE LWO
	0x14E4: "lwoo",        // CANADIAN SYLLABICS LWOO
	0x14E5: "lwoo",        // CANADIAN SYLLABICS WEST-CREE LWOO
	0x14E6: "lwa",         // CANADIAN SYLLABICS LWA
	0x14E7: "lwa",         // CANADIAN SYLLABICS WEST-CREE LWA
	0x14E8: "lwaa",        // CANADIAN SYLLABICS LWAA
	0x14E9: "lwaa",        // CANADIAN SYLLABICS WEST-CREE LWAA
	0x14EA: "l",           // CANADIAN SYLLABICS L
	0x14EB: "l",           // CANADIAN SYLLABICS WEST-CREE L
	0x14EC: "l",           // CANADIAN SYLLABICS MEDIAL L
	0x14ED: "se",          // CANADIAN SYLLABICS SE
	0x14EE: "saai",        // CANADIAN SYLLABICS SAAI
	0x14EF: "si",          // CANADIAN SYLLABICS SI
	0x14F0: "sii",         // CANADIAN SYLLABICS SII
	0x14F1: "so",          // CANADIAN SYLLABICS SO
	0x14F2: "soo",         // CANADIAN SYLLABICS SOO
	0x14F3: "soo",         // CANADIAN SYLLABICS Y-CREE SOO
	0x14F4: "sa",          // CANADIAN SYLLABICS SA
	0x14F5: "saa",         // CANADIAN SYLLABICS SAA
	0x14F6: "swe",         // CANADIAN SYLLABICS SWE
	0x14F7: "swe",         // CANADIAN SYLLABICS WEST-CREE SWE
	0x14F8: "swi",         // CANADIAN SYLLABICS SWI
	0x14F9: "swi",         // CANADIAN SYLLABICS WEST-CREE SWI
	0x14FA: "swii",        // CANADIAN SYLLABICS SWII
	0x14FB: "swii",        // CANADIAN SYLLABICS WEST-CREE SWII
	0x14FC: "swo",         // CANADIAN SYLLABICS SWO
	0x14FD: "swo",         // CANADIAN SYLLABICS WEST-CREE SWO
	0x14FE: "swoo",        // CANADIAN SYLLABICS SWOO
	0x14FF: "swoo",        // CANADIAN SYLLABICS WEST-CREE SWOO
	0x1500: "swa",         // CANADIAN SYLLABICS SWA
	0x1501: "swa",         // CANADIAN SYLLABICS WEST-CREE SWA
	0x1502: "swaa",        // CANADIAN SYLLABICS SWAA
	0x1503: "swaa",        // CANADIAN SYLLABICS WEST-CREE SWAA
	0x1504: "swaa",        // CANADIAN SYLLABICS NASKAPI SWAA
	0x1505: "s",           // CANADIAN SYLLABICS S
	0x1506: "s",           // CANADIAN SYLLABICS ATHAPASCAN S
	0x1507: "sw",          // CANADIAN SYLLABICS SW
	0x1508: "s",           // CANADIAN SYLLABICS BLACKFOOT S
	0x1509: "sk",          // CANADIAN SYLLABICS MOOSE-CREE SK
	0x150A: "skw",         // CANADIAN SYLLABICS NASKAPI SKW
	0x150B: "sW",          // CANADIAN SYLLABICS NASKAPI S-W
	0x150C: "spwa",        // CANADIAN SYLLABICS NASKAPI SPWA
	0x150D: "stwa",        // CANADIAN SYLLABICS NASKAPI STWA
	0x150E: "skwa",        // CANADIAN SYLLABICS NASKAPI SKWA
	0x150F: "scwa",        // CANADIAN SYLLABICS NASKAPI SCWA
	0x1510: "she",         // CANADIAN SYLLABICS SHE
	0x1511: "shi",         // CANADIAN SYLLABICS SHI
	0x1512: "shii",        // CANADIAN SYLLABICS SHII
	0x1513: "sho",         // CANADIAN SYLLABICS SHO
	0x1514: "shoo",        // CANADIAN SYLLABICS SHOO
	0x1515: "sha",         // CANADIAN SYLLABICS SHA
	0x1516: "shaa",        // CANADIAN SYLLABICS SHAA
	0x1517: "shwe",        // CANADIAN SYLLABICS SHWE
	0x1518: "shwe",        // CANADIAN SYLLABICS WEST-CREE SHWE
	0x1519: "shwi",        // CANADIAN SYLLABICS SHWI
	0x151A: "shwi",        // CANADIAN SYLLABICS WEST-CREE SHWI
	0x151B: "shwii",       // CANADIAN SYLLABICS SHWII
	0x151C: "shwii",       // CANADIAN SYLLABICS WEST-CREE SHWII
	0x151D: "shwo",        // CANADIAN SYLLABICS SHWO
	0x151E: "shwo",        // CANADIAN SYLLABICS WEST-CREE SHWO
	0x151F: "shwoo",       // CANADIAN SYLLABICS SHWOO
	0x1520: "shwoo",       // CANADIAN SYLLABICS WEST-CREE SHWOO
	0x1521: "shwa",        // CANADIAN SYLLABICS SHWA
	0x1522: "shwa",        // CANADIAN SYLLABICS WEST-CREE SHWA
	0x1523: "shwaa",       // CANADIAN SYLLABICS SHWAA
	0x1524: "shwaa",       // CANADIAN SYLLABICS WEST-CREE SHWAA
	0x1525: "sh",          // CANADIAN SYLLABICS SH
	0x1526: "ye",          // CANADIAN SYLLABICS YE
	0x1527: "yaai",        // CANADIAN SYLLABICS YAAI
	0x1528: "yi",          // CANADIAN SYLLABICS YI
	0x1529: "yii",         // CANADIAN SYLLABICS YII
	0x152A: "yo",          // CANADIAN SYLLABICS YO
	0x152B: "yoo",         // CANADIAN SYLLABICS YOO
	0x152C: "yoo",         // CANADIAN SYLLABICS Y-CREE YOO
	0x152D: "ya",          // CANADIAN SYLLABICS YA
	0x152E: "yaa",         // CANADIAN SYLLABICS YAA
	0x152F: "ywe",         // CANADIAN SYLLABICS YWE
	0x1530: "ywe",         // CANADIAN SYLLABICS WEST-CREE YWE
	0x1531: "ywi",         // CANADIAN SYLLABICS YWI
	0x1532: "ywi",         // CANADIAN SYLLABICS WEST-CREE YWI
	0x1533: "ywii",        // CANADIAN SYLLABICS YWII
	0x1534: "ywii",        // CANADIAN SYLLABICS WEST-CREE YWII
	0x1535: "ywo",         // CANADIAN SYLLABICS YWO
	0x1536: "ywo",         // CANADIAN SYLLABICS WEST-CREE YWO
	0x1537: "ywoo",        // CANADIAN SYLLABICS YWOO
	0x1538: "ywoo",        // CANADIAN SYLLABICS WEST-CREE YWOO
	0x1539: "ywa",         // CANADIAN SYLLABICS YWA
	0x153A: "ywa",         // CANADIAN SYLLABICS WEST-CREE YWA
	0x153B: "ywaa",        // CANADIAN SYLLABICS YWAA
	0x153C: "ywaa",        // CANADIAN SYLLABICS WEST-CREE YWAA
	0x153D: "ywaa",        // CANADIAN SYLLABICS NASKAPI YWAA
	0x153E: "y",           // CANADIAN SYLLABICS Y
	0x153F: "y",           // CANADIAN SYLLABICS BIBLE-CREE Y
	0x1540: "y",           // CANADIAN SYLLABICS WEST-CREE Y
	0x1541: "yi",          // CANADIAN SYLLABICS SAYISI YI
	0x1542: "re",          // CANADIAN SYLLABICS RE
	0x1543: "re",          // CANADIAN SYLLABICS R-CREE RE
	0x1544: "le",          // CANADIAN SYLLABICS WEST-CREE LE
	0x1545: "raai",        // CANADIAN SYLLABICS RAAI
	0x1546: "ri",          // CANADIAN SYLLABICS RI
	0x1547: "rii",         // CANADIAN SYLLABICS RII
	0x1548: "ro",          // CANADIAN SYLLABICS RO
	0x1549: "roo",         // CANADIAN SYLLABICS ROO
	0x154A: "lo",          // CANADIAN SYLLABICS WEST-CREE LO
	0x154B: "ra",          // CANADIAN SYLLABICS RA
	0x154C: "raa",         // CANADIAN SYLLABICS RAA
	0x154D: "la",          // CANADIAN SYLLABICS WEST-CREE LA
	0x154E: "rwaa",        // CANADIAN SYLLABICS RWAA
	0x154F: "rwaa",        // CANADIAN SYLLABICS WEST-CREE RWAA
	0x1550: "r",           // CANADIAN SYLLABICS R
	0x1551: "r",           // CANADIAN SYLLABICS WEST-CREE R
	0x1552: "r",           // CANADIAN SYLLABICS MEDIAL R
	0x1553: "fe",          // CANADIAN SYLLABICS FE
	0x1554: "faai",        // CANADIAN SYLLABICS FAAI
	0x1555: "fi",          // CANADIAN SYLLABICS FI
	0x1556: "fii",         // CANADIAN SYLLABICS FII
	0x1557: "fo",          // CANADIAN SYLLABICS FO
	0x1558: "foo",         // CANADIAN SYLLABICS FOO
	0x1559: "fa",          // CANADIAN SYLLABICS FA
	0x155A: "faa",         // CANADIAN SYLLABICS FAA
	0x155B: "fwaa",        // CANADIAN SYLLABICS FWAA
	0x155C: "fwaa",        // CANADIAN SYLLABICS WEST-CREE FWAA
	0x155D: "f",           // CANADIAN SYLLABICS F
	0x155E: "the",         // CANADIAN SYLLABICS THE
	0x155F: "the",         // CANADIAN SYLLABICS N-CREE THE
	0x1560: "thi",         // CANADIAN SYLLABICS THI
	0x1561: "thi",         // CANADIAN SYLLABICS N-CREE THI
	0x1562: "thii",        // CANADIAN SYLLABICS THII
	0x1563: "thii",        // CANADIAN SYLLABICS N-CREE THII
	0x1564: "tho",         // CANADIAN SYLLABICS THO
	0x1565: "thoo",        // CANADIAN SYLLABICS THOO
	0x1566: "tha",         // CANADIAN SYLLABICS THA
	0x1567: "thaa",        // CANADIAN SYLLABICS THAA
	0x1568: "thwaa",       // CANADIAN SYLLABICS THWAA
	0x1569: "thwaa",       // CANADIAN SYLLABICS WEST-CREE THWAA
	0x156A: "th",          // CANADIAN SYLLABICS TH
	0x156B: "tthe",        // CANADIAN SYLLABICS TTHE
	0x156C: "tthi",        // CANADIAN SYLLABICS TTHI
	0x156D: "ttho",        // CANADIAN SYLLABICS TTHO
	0x156E: "ttha",        // CANADIAN SYLLABICS TTHA
	0x156F: "tth",         // CANADIAN SYLLABICS TTH
	0x1570: "tye",         // CANADIAN SYLLABICS TYE
	0x1571: "tyi",         // CANADIAN SYLLABICS TYI
	0x1572: "tyo",         // CANADIAN SYLLABICS TYO
	0x1573: "tya",         // CANADIAN SYLLABICS TYA
	0x1574: "he",          // CANADIAN SYLLABICS NUNAVIK HE
	0x1575: "hi",          // CANADIAN SYLLABICS NUNAVIK HI
	0x1576: "hii",         // CANADIAN SYLLABICS NUNAVIK HII
	0x1577: "ho",          // CANADIAN SYLLABICS NUNAVIK HO
	0x1578: "hoo",         // CANADIAN SYLLABICS NUNAVIK HOO
	0x1579: "ha",          // CANADIAN SYLLABICS NUNAVIK HA
	0x157A: "haa",         // CANADIAN SYLLABICS NUNAVIK HAA
	0x157B: "h",           // CANADIAN SYLLABICS NUNAVIK H
	0x157C: "h",           // CANADIAN SYLLABICS NUNAVUT H
	0x157D: "hk",          // CANADIAN SYLLABICS HK
	0x157E: "qaai",        // CANADIAN SYLLABICS QAAI
	0x157F: "qi",          // CANADIAN SYLLABICS QI
	0x1580: "qii",         // CANADIAN SYLLABICS QII
	0x1581: "qo",          // CANADIAN SYLLABICS QO
	0x1582: "qoo",         // CANADIAN SYLLABICS QOO
	0x1583: "qa",          // CANADIAN SYLLABICS QA
	0x1584: "qaa",         // CANADIAN SYLLABICS QAA
	0x1585: "q",           // CANADIAN SYLLABICS Q
	0x1586: "tlhe",        // CANADIAN SYLLABICS TLHE
	0x1587: "tlhi",        // CANADIAN SYLLABICS TLHI
	0x1588: "tlho",        // CANADIAN SYLLABICS TLHO
	0x1589: "tlha",        // CANADIAN SYLLABICS TLHA
	0x158A: "re",          // CANADIAN SYLLABICS WEST-CREE RE
	0x158B: "ri",          // CANADIAN SYLLABICS WEST-CREE RI
	0x158C: "ro",          // CANADIAN SYLLABICS WEST-CREE RO
	0x158D: "ra",          // CANADIAN SYLLABICS WEST-CREE RA
	0x158E: "ngaai",       // CANADIAN SYLLABICS NGAAI
	0x158F: "ngi",         // CANADIAN SYLLABICS NGI
	0x1590: "ngii",        // CANADIAN SYLLABICS NGII
	0x1591: "ngo",         // CANADIAN SYLLABICS NGO
	0x1592: "ngoo",        // CANADIAN SYLLABICS NGOO
	0x1593: "nga",         // CANADIAN SYLLABICS NGA
	0x1594: "ngaa",        // CANADIAN SYLLABICS NGAA
	0x1595: "ng",          // CANADIAN SYLLABICS NG
	0x1596: "nng",         // CANADIAN SYLLABICS NNG
	0x1597: "she",         // CANADIAN SYLLABICS SAYISI SHE
	0x1598: "shi",         // CANADIAN SYLLABICS SAYISI SHI
	0x1599: "sho",         // CANADIAN SYLLABICS SAYISI SHO
	0x159A: "sha",         // CANADIAN SYLLABICS SAYISI SHA
	0x159B: "the",         // CANADIAN SYLLABICS WOODS-CREE THE
	0x159C: "thi",         // CANADIAN SYLLABICS WOODS-CREE THI
	0x159D: "tho",         // CANADIAN SYLLABICS WOODS-CREE THO
	0x159E: "tha",         // CANADIAN SYLLABICS WOODS-CREE THA
	0x159F: "th",          // CANADIAN SYLLABICS WOODS-CREE TH
	0x15A0: "lhi",         // CANADIAN SYLLABICS LHI
	0x15A1: "lhii",        // CANADIAN SYLLABICS LHII
	0x15A2: "lho",         // CANADIAN SYLLABICS LHO
	0x15A3: "lhoo",        // CANADIAN SYLLABICS LHOO
	0x15A4: "lha",         // CANADIAN SYLLABICS LHA
	0x15A5: "lhaa",        // CANADIAN SYLLABICS LHAA
	0x15A6: "lh",          // CANADIAN SYLLABICS LH
	0x15A7: "the",         // CANADIAN SYLLABICS TH-CREE THE
	0x15A8: "thi",         // CANADIAN SYLLABICS TH-CREE THI
	0x15A9: "thii",        // CANADIAN SYLLABICS TH-CREE THII
	0x15AA: "tho",         // CANADIAN SYLLABICS TH-CREE THO
	0x15AB: "thoo",        // CANADIAN SYLLABICS TH-CREE THOO
	0x15AC: "tha",         // CANADIAN SYLLABICS TH-CREE THA
	0x15AD: "thaa",        // CANADIAN SYLLABICS TH-CREE THAA
	0x15AE: "th",          // CANADIAN SYLLABICS TH-CREE TH
	0x15AF: "b",           // CANADIAN SYLLABICS AIVILIK B
	0x15B0: "e",           // CANADIAN SYLLABICS BLACKFOOT E
	0x15B1: "i",           // CANADIAN SYLLABICS BLACKFOOT I
	0x15B2: "o",           // CANADIAN SYLLABICS BLACKFOOT O
	0x15B3: "a",           // CANADIAN SYLLABICS BLACKFOOT A
	0x15B4: "we",          // CANADIAN SYLLABICS BLACKFOOT WE
	0x15B5: "wi",          // CANADIAN SYLLABICS BLACKFOOT WI
	0x15B6: "wo",          // CANADIAN SYLLABICS BLACKFOOT WO
	0x15B7: "wa",          // CANADIAN SYLLABICS BLACKFOOT WA
	0x15B8: "ne",          // CANADIAN SYLLABICS BLACKFOOT NE
	0x15B9: "ni",          // CANADIAN SYLLABICS BLACKFOOT NI
	0x15BA: "no",          // CANADIAN SYLLABICS BLACKFOOT NO
	0x15BB: "na",          // CANADIAN SYLLABICS BLACKFOOT NA
	0x15BC: "ke",          // CANADIAN SYLLABICS BLACKFOOT KE
	0x15BD: "ki",          // CANADIAN SYLLABICS BLACKFOOT KI
	0x15BE: "ko",          // CANADIAN SYLLABICS BLACKFOOT KO
	0x15BF: "ka",          // CANADIAN SYLLABICS BLACKFOOT KA
	0x15C0: "he",          // CANADIAN SYLLABICS SAYISI HE
	0x15C1: "hi",          // CANADIAN SYLLABICS SAYISI HI
	0x15C2: "ho",          // CANADIAN SYLLABICS SAYISI HO
	0x15C3: "ha",          // CANADIAN SYLLABICS SAYISI HA
	0x15C4: "ghu",         // CANADIAN SYLLABICS CARRIER GHU
	0x15C5: "gho",         // CANADIAN SYLLABICS CARRIER GHO
	0x15C6: "ghe",         // CANADIAN SYLLABICS CARRIER GHE
	0x15C7: "ghee",        // CANADIAN SYLLABICS CARRIER GHEE
	0x15C8: "ghi",         // CANADIAN SYLLABICS CARRIER GHI
	0x15C9: "gha",         // CANADIAN SYLLABICS CARRIER GHA
	0x15CA: "ru",          // CANADIAN SYLLABICS CARRIER RU
	0x15CB: "ro",          // CANADIAN SYLLABICS CARRIER RO
	0x15CC: "re",          // CANADIAN SYLLABICS CARRIER RE
	0x15CD: "ree",         // CANADIAN SYLLABICS CARRIER REE
	0x15CE: "ri",          // CANADIAN SYLLABICS CARRIER RI
	0x15CF: "ra",          // CANADIAN SYLLABICS CARRIER RA
	0x15D0: "wu",          // CANADIAN SYLLABICS CARRIER WU
	0x15D1: "wo",          // CANADIAN SYLLABICS CARRIER WO
	0x15D2: "we",          // CANADIAN SYLLABICS CARRIER WE
	0x15D3: "wee",         // CANADIAN SYLLABICS CARRIER WEE
	0x15D4: "wi",          // CANADIAN SYLLABICS CARRIER WI
	0x15D5: "wa",          // CANADIAN SYLLABICS CARRIER WA
	0x15D6: "hwu",         // CANADIAN SYLLABICS CARRIER HWU
	0x15D7: "hwo",         // CANADIAN SYLLABICS CARRIER HWO
	0x15D8: "hwe",         // CANADIAN SYLLABICS CARRIER HWE
	0x15D9: "hwee",        // CANADIAN SYLLABICS CARRIER HWEE
	0x15DA: "hwi",         // CANADIAN SYLLABICS CARRIER HWI
	0x15DB: "hwa",         // CANADIAN SYLLABICS CARRIER HWA
	0x15DC: "thu",         // CANADIAN SYLLABICS CARRIER THU
	0x15DD: "tho",         // CANADIAN SYLLABICS CARRIER THO
	0x15DE: "the",         // CANADIAN SYLLABICS CARRIER THE
	0x15DF: "thee",        // CANADIAN SYLLABICS CARRIER THEE
	0x15E0: "thi",         // CANADIAN SYLLABICS CARRIER THI
	0x15E1: "tha",         // CANADIAN SYLLABICS CARRIER THA
	0x15E2: "ttu",         // CANADIAN SYLLABICS CARRIER TTU
	0x15E3: "tto",         // CANADIAN SYLLABICS CARRIER TTO
	0x15E4: "tte",         // CANADIAN SYLLABICS CARRIER TTE
	0x15E5: "ttee",        // CANADIAN SYLLABICS CARRIER TTEE
	0x15E6: "tti",         // CANADIAN SYLLABICS CARRIER TTI
	0x15E7: "tta",         // CANADIAN SYLLABICS CARRIER TTA
	0x15E8: "pu",          // CANADIAN SYLLABICS CARRIER PU
	0x15E9: "po",          // CANADIAN SYLLABICS CARRIER PO
	0x15EA: "pe",          // CANADIAN SYLLABICS CARRIER PE
	0x15EB: "pee",         // CANADIAN SYLLABICS CARRIER PEE
	0x15EC: "pi",          // CANADIAN SYLLABICS CARRIER PI
	0x15ED: "pa",          // CANADIAN SYLLABICS CARRIER PA
	0x15EE: "p",           // CANADIAN SYLLABICS CARRIER P
	0x15EF: "gu",          // CANADIAN SYLLABICS CARRIER GU
	0x15F0: "go",          // CANADIAN SYLLABICS CARRIER GO
	0x15F1: "ge",          // CANADIAN SYLLABICS CARRIER GE
	0x15F2: "gee",         // CANADIAN SYLLABICS CARRIER GEE
	0x15F3: "gi",          // CANADIAN SYLLABICS CARRIER GI
	0x15F4: "ga",          // CANADIAN SYLLABICS CARRIER GA
	0x15F5: "khu",         // CANADIAN SYLLABICS CARRIER KHU
	0x15F6: "kho",         // CANADIAN SYLLABICS CARRIER KHO
	0x15F7: "khe",         // CANADIAN SYLLABICS CARRIER KHE
	0x15F8: "khee",        // CANADIAN SYLLABICS CARRIER KHEE
	0x15F9: "khi",         // CANADIAN SYLLABICS CARRIER KHI
	0x15FA: "kha",         // CANADIAN SYLLABICS CARRIER KHA
	0x15FB: "kku",         // CANADIAN SYLLABICS CARRIER KKU
	0x15FC: "kko",         // CANADIAN SYLLABICS CARRIER KKO
	0x15FD: "kke",         // CANADIAN SYLLABICS CARRIER KKE
	0x15FE: "kkee",        // CANADIAN SYLLABICS CARRIER KKEE
	0x15FF: "kki",         // CANADIAN SYLLABICS CARRIER KKI
	0x1600: "kka",         // CANADIAN SYLLABICS CARRIER KKA
	0x1601: "kk",          // CANADIAN SYLLABICS CARRIER KK
	0x1602: "nu",          // CANADIAN SYLLABICS CARRIER NU
	0x1603: "no",          // CANADIAN SYLLABICS CARRIER NO
	0x1604: "ne",          // CANADIAN SYLLABICS CARRIER NE
	0x1605: "nee",         // CANADIAN SYLLABICS CARRIER NEE
	0x1606: "ni",          // CANADIAN SYLLABICS CARRIER NI
	0x1607: "na",          // CANADIAN SYLLABICS CARRIER NA
	0x1608: "mu",          // CANADIAN SYLLABICS CARRIER MU
	0x1609: "mo",          // CANADIAN SYLLABICS CARRIER MO
	0x160A: "me",          // CANADIAN SYLLABICS CARRIER ME
	0x160B: "mee",         // CANADIAN SYLLABICS CARRIER MEE
	0x160C: "mi",          // CANADIAN SYLLABICS CARRIER MI
	0x160D: "ma",          // CANADIAN SYLLABICS CARRIER MA
	0x160E: "yu",          // CANADIAN SYLLABICS CARRIER YU
	0x160F: "yo",          // CANADIAN SYLLABICS CARRIER YO
	0x1610: "ye",          // CANADIAN SYLLABICS CARRIER YE
	0x1611: "yee",         // CANADIAN SYLLABICS CARRIER YEE
	0x1612: "yi",          // CANADIAN SYLLABICS CARRIER YI
	0x1613: "ya",          // CANADIAN SYLLABICS CARRIER YA
	0x1614: "ju",          // CANADIAN SYLLABICS CARRIER JU
	0x1615: "ju",          // CANADIAN SYLLABICS SAYISI JU
	0x1616: "jo",          // CANADIAN SYLLABICS CARRIER JO
	0x1617: "je",          // CANADIAN SYLLABICS CARRIER JE
	0x1618: "jee",         // CANADIAN SYLLABICS CARRIER JEE
	0x1619: "ji",          // CANADIAN SYLLABICS CARRIER JI
	0x161A: "ji",          // CANADIAN SYLLABICS SAYISI JI
	0x161B: "ja",          // CANADIAN SYLLABICS CARRIER JA
	0x161C: "jju",         // CANADIAN SYLLABICS CARRIER JJU
	0x161D: "jjo",         // CANADIAN SYLLABICS CARRIER JJO
	0x161E: "jje",         // CANADIAN SYLLABICS CARRIER JJE
	0x161F: "jjee",        // CANADIAN SYLLABICS CARRIER JJEE
	0x1620: "jji",         // CANADIAN SYLLABICS CARRIER JJI
	0x1621: "jja",         // CANADIAN SYLLABICS CARRIER JJA
	0x1622: "lu",          // CANADIAN SYLLABICS CARRIER LU
	0x1623: "lo",          // CANADIAN SYLLABICS CARRIER LO
	0x1624: "le",          // CANADIAN SYLLABICS CARRIER LE
	0x1625: "lee",         // CANADIAN SYLLABICS CARRIER LEE
	0x1626: "li",          // CANADIAN SYLLABICS CARRIER LI
	0x1627: "la",          // CANADIAN SYLLABICS CARRIER LA
	0x1628: "dlu",         // CANADIAN SYLLABICS CARRIER DLU
	0x1629: "dlo",         // CANADIAN SYLLABICS CARRIER DLO
	0x162A: "dle",         // CANADIAN SYLLABICS CARRIER DLE
	0x162B: "dlee",        // CANADIAN SYLLABICS CARRIER DLEE
	0x162C: "dli",         // CANADIAN SYLLABICS CARRIER DLI
	0x162D: "dla",         // CANADIAN SYLLABICS CARRIER DLA
	0x162E: "lhu",         // CANADIAN SYLLABICS CARRIER LHU
	0x162F: "lho",         // CANADIAN SYLLABICS CARRIER LHO
	0x1630: "lhe",         // CANADIAN SYLLABICS CARRIER LHE
	0x1631: "lhee",        // CANADIAN SYLLABICS CARRIER LHEE
	0x1632: "lhi",         // CANADIAN SYLLABICS CARRIER LHI
	0x1633: "lha",         // CANADIAN SYLLABICS CARRIER LHA
	0x1634: "tlhu",        // CANADIAN SYLLABICS CARRIER TLHU
	0x1635: "tlho",        // CANADIAN SYLLABICS CARRIER TLHO
	0x1636: "tlhe",        // CANADIAN SYLLABICS CARRIER TLHE
	0x1637: "tlhee",       // CANADIAN SYLLABICS CARRIER TLHEE
	0x1638: "tlhi",        // CANADIAN SYLLABICS CARRIER TLHI
	0x1639: "tlha",        // CANADIAN SYLLABICS CARRIER TLHA
	0x163A: "tlu",         // CANADIAN SYLLABICS CARRIER TLU
	0x163B: "tlo",         // CANADIAN SYLLABICS CARRIER TLO
	0x163C: "tle",         // CANADIAN SYLLABICS CARRIER TLE
	0x163D: "tlee",        // CANADIAN SYLLABICS CARRIER TLEE
	0x163E: "tli",         // CANADIAN SYLLABICS CARRIER TLI
	0x163F: "tla",         // CANADIAN SYLLABICS CARRIER TLA
	0x1640: "zu",          // CANADIAN SYLLABICS CARRIER ZU
	0x1641: "zo",          // CANADIAN SYLLABICS CARRIER ZO
	0x1642: "ze",          // CANADIAN SYLLABICS CARRIER ZE
	0x1643: "zee",         // CANADIAN SYLLABICS CARRIER ZEE
	0x1644: "zi",          // CANADIAN SYLLABICS CARRIER ZI
	0x1645: "za",          // CANADIAN SYLLABICS CARRIER ZA
	0x1646: "z",           // CANADIAN SYLLABICS CARRIER Z
	0x1647: "z",           // CANADIAN SYLLABICS CARRIER INITIAL Z
	0x1648: "dzu",         // CANADIAN SYLLABICS CARRIER DZU
	0x1649: "dzo",         // CANADIAN SYLLABICS CARRIER DZO
	0x164A: "dze",         // CANADIAN SYLLABICS CARRIER DZE
	0x164B: "dzee",        // CANADIAN SYLLABICS CARRIER DZEE
	0x164C: "dzi",         // CANADIAN SYLLABICS CARRIER DZI
	0x164D: "dza",         // CANADIAN SYLLABICS CARRIER DZA
	0x164E: "su",          // CANADIAN SYLLABICS CARRIER SU
	0x164F: "so",          // CANADIAN SYLLABICS CARRIER SO
	0x1650: "se",          // CANADIAN SYLLABICS CARRIER SE
	0x1651: "see",         // CANADIAN SYLLABICS CARRIER SEE
	0x1652: "si",          // CANADIAN SYLLABICS CARRIER SI
	0x1653: "sa",          // CANADIAN SYLLABICS CARRIER SA
	0x1654: "shu",         // CANADIAN SYLLABICS CARRIER SHU
	0x1655: "sho",         // CANADIAN SYLLABICS CARRIER SHO
	0x1656: "she",         // CANADIAN SYLLABICS CARRIER SHE
	0x1657: "shee",        // CANADIAN SYLLABICS CARRIER SHEE
	0x1658: "shi",         // CANADIAN SYLLABICS CARRIER SHI
	0x1659: "sha",         // CANADIAN SYLLABICS CARRIER SHA
	0x165A: "sh",          // CANADIAN SYLLABICS CARRIER SH
	0x165B: "tsu",         // CANADIAN SYLLABICS CARRIER TSU
	0x165C: "tso",         // CANADIAN SYLLABICS CARRIER TSO
	0x165D: "tse",         // CANADIAN SYLLABICS CARRIER TSE
	0x165E: "tsee",        // CANADIAN SYLLABICS CARRIER TSEE
	0x165F: "tsi",         // CANADIAN SYLLABICS CARRIER TSI
	0x1660: "tsa",         // CANADIAN SYLLABICS CARRIER TSA
	0x1661: "chu",         // CANADIAN SYLLABICS CARRIER CHU
	0x1662: "cho",         // CANADIAN SYLLABICS CARRIER CHO
	0x1663: "che",         // CANADIAN SYLLABICS CARRIER CHE
	0x1664: "chee",        // CANADIAN SYLLABICS CARRIER CHEE
	0x1665: "chi",         // CANADIAN SYLLABICS CARRIER CHI
	0x1666: "cha",         // CANADIAN SYLLABICS CARRIER CHA
	0x1667: "ttsu",        // CANADIAN SYLLABICS CARRIER TTSU
	0x1668: "ttso",        // CANADIAN SYLLABICS CARRIER TTSO
	0x1669: "ttse",        // CANADIAN SYLLABICS CARRIER TTSE
	0x166A: "ttsee",       // CANADIAN SYLLABICS CARRIER TTSEE
	0x166B: "ttsi",        // CANADIAN SYLLABICS CARRIER TTSI
	0x166C: "ttsa",        // CANADIAN SYLLABICS CARRIER TTSA
	0x166D: "X",           // CANADIAN SYLLABICS CHI SIGN
	0x166E: ".",           // CANADIAN SYLLABICS FULL STOP
	0x166F: "qai",         // CANADIAN SYLLABICS QAI
	0x1670: "ngai",        // CANADIAN SYLLABICS NGAI
	0x1671: "nngi",        // CANADIAN SYLLABICS NNGI
	0x1672: "nngii",       // CANADIAN SYLLABICS NNGII
	0x1673: "nngo",        // CANADIAN SYLLABICS NNGO
	0x1674: "nngoo",       // CANADIAN SYLLABICS NNGOO
	0x1675: "nnga",        // CANADIAN SYLLABICS NNGA
	0x1676: "nngaa",       // CANADIAN SYLLABICS NNGAA
	0x1681: "b",           // OGHAM LETTER BEITH
	0x1682: "l",           // OGHAM LETTER LUIS
	0x1683: "f",           // OGHAM LETTER FEARN
	0x1684: "s",           // OGHAM LETTER SAIL
	0x1685: "n",           // OGHAM LETTER NION
	0x1686: "h",           // OGHAM LETTER UATH
	0x1687: "d",           // OGHAM LETTER DAIR
	0x1688: "t",           // OGHAM LETTER TINNE
	0x1689: "c",           // OGHAM LETTER COLL
	0x168A: "q",           // OGHAM LETTER CEIRT
	0x168B: "m",           // OGHAM LETTER MUIN
	0x168C: "g",           // OGHAM LETTER GORT
	0x168D: "ng",          // OGHAM LETTER NGEADAL
	0x168E: "z",           // OGHAM LETTER STRAIF
	0x168F: "r",           // OGHAM LETTER RUIS
	0x1690: "a",           // OGHAM LETTER AILM
	0x1691: "o",           // OGHAM LETTER ONN
	0x1692: "u",           // OGHAM LETTER UR
	0x1693: "e",           // OGHAM LETTER EADHADH
	0x1694: "i",           // OGHAM LETTER IODHADH
	0x1695: "ch",          // OGHAM LETTER EABHADH
	0x1696: "th",          // OGHAM LETTER OR
	0x1697: "ph",          // OGHAM LETTER UILLEANN
	0x1698: "p",           // OGHAM LETTER IFIN
	0x1699: "x",           // OGHAM LETTER EAMHANCHOLL
	0x169A: "p",           // OGHAM LETTER PEITH
	0x169B: "<",           // OGHAM FEATHER MARK
	0x169C: ">",           // OGHAM REVERSED FEATHER MARK
	0x16A0: "f",           // RUNIC LETTER FEHU FEOH FE F
	0x16A1: "v",           // RUNIC LETTER V
	0x16A2: "u",           // RUNIC LETTER URUZ UR U
	0x16A3: "yr",          // RUNIC LETTER YR
	0x16A4: "y",           // RUNIC LETTER Y
	0x16A5: "w",           // RUNIC LETTER W
	0x16A6: "th",          // RUNIC LETTER THURISAZ THURS THORN
	0x16A7: "th",          // RUNIC LETTER ETH
	0x16A8: "a",           // RUNIC LETTER ANSUZ A
	0x16A9: "o",           // RUNIC LETTER OS O
	0x16AA: "ac",          // RUNIC LETTER AC A
	0x16AB: "ae",          // RUNIC LETTER AESC
	0x16AC: "o",           // RUNIC LETTER LONG-BRANCH-OSS O
	0x16AD: "o",           // RUNIC LETTER SHORT-TWIG-OSS O
	0x16AE: "o",           // RUNIC LETTER O
	0x16AF: "oe",          // RUNIC LETTER OE
	0x16B0: "on",          // RUNIC LETTER ON
	0x16B1: "r",           // RUNIC LETTER RAIDO RAD REID R
	0x16B2: "k",           // RUNIC LETTER KAUNA
	0x16B3: "c",           // RUNIC LETTER CEN
	0x16B4: "k",           // RUNIC LETTER KAUN K
	0x16B5: "g",           // RUNIC LETTER G
	0x16B6: "ng",          // RUNIC LETTER ENG
	0x16B7: "g",           // RUNIC LETTER GEBO GYFU G
	0x16B8: "g",           // RUNIC LETTER GAR
	0x16B9: "w",           // RUNIC LETTER WUNJO WYNN W
	0x16BA: "h",           // RUNIC LETTER HAGLAZ H
	0x16BB: "h",           // RUNIC LETTER HAEGL H
	0x16BC: "h",           // RUNIC LETTER LONG-BRANCH-HAGALL H
	0x16BD: "h",           // RUNIC LETTER SHORT-TWIG-HAGALL H
	0x16BE: "n",           // RUNIC LETTER NAUDIZ NYD NAUD N
	0x16BF: "n",           // RUNIC LETTER SHORT-TWIG-NAUD N
	0x16C0: "n",           // RUNIC LETTER DOTTED-N
	0x16C1: "i",           // RUNIC LETTER ISAZ IS ISS I
	0x16C2: "e",           // RUNIC LETTER E
	0x16C3: "j",           // RUNIC LETTER JERAN J
	0x16C4: "g",           // RUNIC LETTER GER
	0x16C5: "ae",          // RUNIC LETTER LONG-BRANCH-AR AE
	0x16C6: "a",           // RUNIC LETTER SHORT-TWIG-AR A
	0x16C7: "eo",          // RUNIC LETTER IWAZ EOH
	0x16C8: "p",           // RUNIC LETTER PERTHO PEORTH P
	0x16C9: "z",           // RUNIC LETTER ALGIZ EOLHX
	0x16CA: "s",           // RUNIC LETTER SOWILO S
	0x16CB: "s",           // RUNIC LETTER SIGEL LONG-BRANCH-SOL S
	0x16CC: "s",           // RUNIC LETTER SHORT-TWIG-SOL S
	0x16CD: "c",           // RUNIC LETTER C
	0x16CE: "z",           // RUNIC LETTER Z
	0x16CF: "t",           // RUNIC LETTER TIWAZ TIR TYR T
	0x16D0: "t",           // RUNIC LETTER SHORT-TWIG-TYR T
	0x16D1: "d",           // RUNIC LETTER D
	0x16D2: "b",           // RUNIC LETTER BERKANAN BEORC BJARKAN B
	0x16D3: "b",           // RUNIC LETTER SHORT-TWIG-BJARKAN B
	0x16D4: "p",           // RUNIC LETTER DOTTED-P
	0x16D5: "p",           // RUNIC LETTER OPEN-P
	0x16D6: "e",           // RUNIC LETTER EHWAZ EH E
	0x16D7: "m",           // RUNIC LETTER MANNAZ MAN M
	0x16D8: "m",           // RUNIC LETTER LONG-BRANCH-MADR M
	0x16D9: "m",           // RUNIC LETTER SHORT-TWIG-MADR M
	0x16DA: "l",           // RUNIC LETTER LAUKAZ LAGU LOGR L
	0x16DB: "l",           // RUNIC LETTER DOTTED-L
	0x16DC: "ng",          // RUNIC LETTER INGWAZ
	0x16DD: "ng",          // RUNIC LETTER ING
	0x16DE: "d",           // RUNIC LETTER DAGAZ DAEG D
	0x16DF: "o",           // RUNIC LETTER OTHALAN ETHEL O
	0x16E0: "ear",         // RUNIC LETTER EAR
	0x16E1: "ior",         // RUNIC LETTER IOR
	0x16E2: "qu",          // RUNIC LETTER CWEORTH
	0x16E3: "qu",          // RUNIC LETTER CALC
	0x16E4: "qu",          // RUNIC LETTER CEALC
	0x16E5: "s",           // RUNIC LETTER STAN
	0x16E6: "yr",          // RUNIC LETTER LONG-BRANCH-YR
	0x16E7: "yr",          // RUNIC LETTER SHORT-TWIG-YR
	0x16E8: "yr",          // RUNIC LETTER ICELANDIC-YR
	0x16E9: "q",           // RUNIC LETTER Q
	0x16EA: "x",           // RUNIC LETTER X
	0x16EB: ".",           // RUNIC SINGLE PUNCTUATION
	0x16EC: ":",           // RUNIC MULTIPLE PUNCTUATION
	0x16ED: "+",           // RUNIC CROSS PUNCTUATION
	0x16EE: "17",          // RUNIC ARLAUG SYMBOL
	0x16EF: "18",          // RUNIC TVIMADUR SYMBOL
	0x16F0: "19",          // RUNIC BELGTHOR SYMBOL
	0x1780: "k",           // KHMER LETTER KA
	0x1781: "kh",          // KHMER LETTER KHA
	0x1782: "g",           // KHMER LETTER KO
	0x1783: "gh",          // KHMER LETTER KHO
	0x1784: "ng",          // KHMER LETTER NGO
	0x1785: "c",           // KHMER LETTER CA
	0x1786: "ch",          // KHMER LETTER CHA
	0x1787: "j",           // KHMER LETTER CO
	0x1788: "jh",          // KHMER LETTER CHO
	0x1789: "ny",          // KHMER LETTER NYO
	0x178A: "t",           // KHMER LETTER DA
	0x178B: "tth",         // KHMER LETTER TTHA
	0x178C: "d",           // KHMER LETTER DO
	0x178D: "ddh",         // KHMER LETTER TTHO
	0x178E: "nn",          // KHMER LETTER NNO
	0x178F: "t",           // KHMER LETTER TA
	0x1790: "th",          // KHMER LETTER THA
	0x1791: "d",           // KHMER LETTER TO
	0x1792: "dh",          // KHMER LETTER THO
	0x1793: "n",           // KHMER LETTER NO
	0x1794: "p",           // KHMER LETTER BA
	0x1795: "ph",          // KHMER LETTER PHA
	0x1796: "b",           // KHMER LETTER PO
	0x1797: "bh",          // KHMER LETTER PHO
	0x1798: "m",           // KHMER LETTER MO
	0x1799: "y",           // KHMER LETTER YO
	0x179A: "r",           // KHMER LETTER RO
	0x179B: "l",           // KHMER LETTER LO
	0x179C: "v",           // KHMER LETTER VO
	0x179D: "sh",          // KHMER LETTER SHA
	0x179E: "ss",          // KHMER LETTER SSO
	0x179F: "s",           // KHMER LETTER SA
	0x17A0: "h",           // KHMER LETTER HA
	0x17A1: "l",           // KHMER LETTER LA
	0x17A2: "q",           // KHMER LETTER QA
	0x17A3: "a",           // KHMER INDEPENDENT VOWEL QAQ
	0x17A4: "aa",          // KHMER INDEPENDENT VOWEL QAA
	0x17A5: "i",           // KHMER INDEPENDENT VOWEL QI
	0x17A6: "ii",          // KHMER INDEPENDENT VOWEL QII
	0x17A7: "u",           // KHMER INDEPENDENT VOWEL QU
	0x17A8: "uk",          // KHMER INDEPENDENT VOWEL QUK
	0x17A9: "uu",          // KHMER INDEPENDENT VOWEL QUU
	0x17AA: "uuv",         // KHMER INDEPENDENT VOWEL QUUV
	0x17AB: "ry",          // KHMER INDEPENDENT VOWEL RY
	0x17AC: "ryy",         // KHMER INDEPENDENT VOWEL RYY
	0x17AD: "ly",          // KHMER INDEPENDENT VOWEL LY
	0x17AE: "lyy",         // KHMER INDEPENDENT VOWEL LYY
	0x17AF: "e",           // KHMER INDEPENDENT VOWEL QE
	0x17B0: "ai",          // KHMER INDEPENDENT VOWEL QAI
	0x17B1: "oo",          // KHMER INDEPENDENT VOWEL QOO TYPE ONE
	0x17B2: "oo",          // KHMER INDEPENDENT VOWEL QOO TYPE TWO
	0x17B3: "au",          // KHMER INDEPENDENT VOWEL QAU
	0x17B4: "a",           // KHMER VOWEL INHERENT AQ
	0x17B5: "aa",          // KHMER VOWEL INHERENT AA
	0x17B6: "aa",          // KHMER VOWEL SIGN AA
	0x17B7: "i",           // KHMER VOWEL SIGN I
	0x17B8: "ii",          // KHMER VOWEL SIGN II
	0x17B9: "y",           // KHMER VOWEL SIGN Y
	0x17BA: "yy",          // KHMER VOWEL SIGN YY
	0x17BB: "u",           // KHMER VOWEL SIGN U
	0x17BC: "uu",          // KHMER VOWEL SIGN UU
	0x17BD: "ua",          // KHMER VOWEL SIGN UA
	0x17BE: "oe",          // KHMER VOWEL SIGN OE
	0x17BF: "ya",          // KHMER VOWEL SIGN YA
	0x17C0: "ie",          // KHMER VOWEL SIGN IE
	0x17C1: "e",           // KHMER VOWEL SIGN E
	0x17C2: "ae",          // KHMER VOWEL SIGN AE
	0x17C3: "ai",          // KHMER VOWEL SIGN AI
	0x17C4: "oo",          // KHMER VOWEL SIGN OO
	0x17C5: "au",          // KHMER VOWEL SIGN AU
	0x17C6: "M",           // KHMER SIGN NIKAHIT
	0x17C7: "H",           // KHMER SIGN REAHMUK
	0x17C8: "a`",          // KHMER SIGN YUUKALEAPINTU
	0x17CC: "r",           // KHMER SIGN ROBAT
	0x17CE: "!",           // KHMER SIGN KAKABAT
	0x17D4: ".",           // KHMER SIGN KHAN
	0x17D5: " // ",        // KHMER SIGN BARIYOOSAN
	0x17D6: ":",           // KHMER SIGN CAMNUC PII KUUH
	0x17D7: "+",           // KHMER SIGN LEK TOO
	0x17D8: "++",          // KHMER SIGN BEYYAL
	0x17D9: " * ",         // KHMER SIGN PHNAEK MUAN
	0x17DA: " /// ",       // KHMER SIGN KOOMUUT
	0x17DB: "KR",          // KHMER CURRENCY SYMBOL RIEL
	0x17DC: "'",           // KHMER SIGN AVAKRAHASANYA
	0x17E0: "0",           // KHMER DIGIT ZERO
	0x17E1: "1",           // KHMER DIGIT ONE
	0x17E2: "2",           // KHMER DIGIT TWO
	0x17E3: "3",           // KHMER DIGIT THREE
	0x17E4: "4",           // KHMER DIGIT FOUR
	0x17E5: "5",           // KHMER DIGIT FIVE
	0x17E6: "6",           // KHMER DIGIT SIX
	0x17E7: "7",           // KHMER DIGIT SEVEN
	0x17E8: "8",           // KHMER DIGIT EIGHT
	0x17E9: "9",           // KHMER DIGIT NINE
	0x1800: " @ ",         // MONGOLIAN BIRGA
	0x1801: " ... ",       // MONGOLIAN ELLIPSIS
	0x1802: ", ",          // MONGOLIAN COMMA
	0x1803: ". ",          // MONGOLIAN FULL STOP
	0x1804: ": ",          // MONGOLIAN COLON
	0x1805: " // ",        // MONGOLIAN FOUR DOTS
	0x1806: "",            // MONGOLIAN TODO SOFT HYPHEN
	0x1807: "-",           // MONGOLIAN SIBE SYLLABLE BOUNDARY MARKER
	0x1808: ", ",          // MONGOLIAN MANCHU COMMA
	0x1809: ". ",          // MONGOLIAN MANCHU FULL STOP
	0x1810: "0",           // MONGOLIAN DIGIT ZERO
	0x1811: "1",           // MONGOLIAN DIGIT ONE
	0x1812: "2",           // MONGOLIAN DIGIT TWO
	0x1813: "3",           // MONGOLIAN DIGIT THREE
	0x1814: "4",           // MONGOLIAN DIGIT FOUR
	0x1815: "5",           // MONGOLIAN DIGIT FIVE
	0x1816: "6",           // MONGOLIAN DIGIT SIX
	0x1817: "7",           // MONGOLIAN DIGIT SEVEN
	0x1818: "8",           // MONGOLIAN DIGIT EIGHT
	0x1819: "9",           // MONGOLIAN DIGIT NINE
	0x1820: "a",           // MONGOLIAN LETTER A
	0x1821: "e",           // MONGOLIAN LETTER E
	0x1822: "i",           // MONGOLIAN LETTER I
	0x1823: "o",           // MONGOLIAN LETTER O
	0x1824: "u",           // MONGOLIAN LETTER U
	0x1825: "O",           // MONGOLIAN LETTER OE
	0x1826: "U",           // MONGOLIAN LETTER UE
	0x1827: "ee",          // MONGOLIAN LETTER EE
	0x1828: "n",           // MONGOLIAN LETTER NA
	0x1829: "ng",          // MONGOLIAN LETTER ANG
	0x182A: "b",           // MONGOLIAN LETTER BA
	0x182B: "p",           // MONGOLIAN LETTER PA
	0x182C: "q",           // MONGOLIAN LETTER QA
	0x182D: "g",           // MONGOLIAN LETTER GA
	0x182E: "m",           // MONGOLIAN LETTER MA
	0x182F: "l",           // MONGOLIAN LETTER LA
	0x1830: "s",           // MONGOLIAN LETTER SA
	0x1831: "sh",          // MONGOLIAN LETTER SHA
	0x1832: "t",           // MONGOLIAN LETTER TA
	0x1833: "d",           // MONGOLIAN LETTER DA
	0x1834: "ch",          // MONGOLIAN LETTER CHA
	0x1835: "j",           // MONGOLIAN LETTER JA
	0x1836: "y",           // MONGOLIAN LETTER YA
	0x1837: "r",           // MONGOLIAN LETTER RA
	0x1838: "w",           // MONGOLIAN LETTER WA
	0x1839: "f",           // MONGOLIAN LETTER FA
	0x183A: "k",           // MONGOLIAN LETTER KA
	0x183B: "kha",         // MONGOLIAN LETTER KHA
	0x183C: "ts",          // MONGOLIAN LETTER TSA
	0x183D: "z",           // MONGOLIAN LETTER ZA
	0x183E: "h",           // MONGOLIAN LETTER HAA
	0x183F: "zr",          // MONGOLIAN LETTER ZRA
	0x1840: "lh",          // MONGOLIAN LETTER LHA
	0x1841: "zh",          // MONGOLIAN LETTER ZHI
	0x1842: "ch",          // MONGOLIAN LETTER CHI
	0x1843: "-",           // MONGOLIAN LETTER TODO LONG VOWEL SIGN
	0x1844: "e",           // MONGOLIAN LETTER TODO E
	0x1845: "i",           // MONGOLIAN LETTER TODO I
	0x1846: "o",           // MONGOLIAN LETTER TODO O
	0x1847: "u",           // MONGOLIAN LETTER TODO U
	0x1848: "O",           // MONGOLIAN LETTER TODO OE
	0x1849: "U",           // MONGOLIAN LETTER TODO UE
	0x184A: "ng",          // MONGOLIAN LETTER TODO ANG
	0x184B: "b",           // MONGOLIAN LETTER TODO BA
	0x184C: "p",           // MONGOLIAN LETTER TODO PA
	0x184D: "q",           // MONGOLIAN LETTER TODO QA
	0x184E: "g",           // MONGOLIAN LETTER TODO GA
	0x184F: "m",           // MONGOLIAN LETTER TODO MA
	0x1850: "t",           // MONGOLIAN LETTER TODO TA
	0x1851: "d",           // MONGOLIAN LETTER TODO DA
	0x1852: "ch",          // MONGOLIAN LETTER TODO CHA
	0x1853: "j",           // MONGOLIAN LETTER TODO JA
	0x1854: "ts",          // MONGOLIAN LETTER TODO TSA
	0x1855: "y",           // MONGOLIAN LETTER TODO YA
	0x1856: "w",           // MONGOLIAN LETTER TODO WA
	0x1857: "k",           // MONGOLIAN LETTER TODO KA
	0x1858: "g",           // MONGOLIAN LETTER TODO GAA
	0x1859: "h",           // MONGOLIAN LETTER TODO HAA
	0x185A: "jy",          // MONGOLIAN LETTER TODO JIA
	0x185B: "ny",          // MONGOLIAN LETTER TODO NIA
	0x185C: "dz",          // MONGOLIAN LETTER TODO DZA
	0x185D: "e",           // MONGOLIAN LETTER SIBE E
	0x185E: "i",           // MONGOLIAN LETTER SIBE I
	0x185F: "iy",          // MONGOLIAN LETTER SIBE IY
	0x1860: "U",           // MONGOLIAN LETTER SIBE UE
	0x1861: "u",           // MONGOLIAN LETTER SIBE U
	0x1862: "ng",          // MONGOLIAN LETTER SIBE ANG
	0x1863: "k",           // MONGOLIAN LETTER SIBE KA
	0x1864: "g",           // MONGOLIAN LETTER SIBE GA
	0x1865: "h",           // MONGOLIAN LETTER SIBE HA
	0x1866: "p",           // MONGOLIAN LETTER SIBE PA
	0x1867: "sh",          // MONGOLIAN LETTER SIBE SHA
	0x1868: "t",           // MONGOLIAN LETTER SIBE TA
	0x1869: "d",           // MONGOLIAN LETTER SIBE DA
	0x186A: "j",           // MONGOLIAN LETTER SIBE JA
	0x186B: "f",           // MONGOLIAN LETTER SIBE FA
	0x186C: "g",           // MONGOLIAN LETTER SIBE GAA
	0x186D: "h",           // MONGOLIAN LETTER SIBE HAA
	0x186E: "ts",          // MONGOLIAN LETTER SIBE TSA
	0x186F: "z",           // MONGOLIAN LETTER SIBE ZA
	0x1870: "r",           // MONGOLIAN LETTER SIBE RAA
	0x1871: "ch",          // MONGOLIAN LETTER SIBE CHA
	0x1872: "zh",          // MONGOLIAN LETTER SIBE ZHA
	0x1873: "i",           // MONGOLIAN LETTER MANCHU I
	0x1874: "k",           // MONGOLIAN LETTER MANCHU KA
	0x1875: "r",           // MONGOLIAN LETTER MANCHU RA
	0x1876: "f",           // MONGOLIAN LETTER MANCHU FA
	0x1877: "zh",          // MONGOLIAN LETTER MANCHU ZHA
	0x1881: "H",           // MONGOLIAN LETTER ALI GALI VISARGA ONE
	0x1882: "X",           // MONGOLIAN LETTER ALI GALI DAMARU
	0x1883: "W",           // MONGOLIAN LETTER ALI GALI UBADAMA
	0x1884: "M",           // MONGOLIAN LETTER ALI GALI INVERTED UBADAMA
	0x1885: " 3 ",         // MONGOLIAN LETTER ALI GALI BALUDA
	0x1886: " 333 ",       // MONGOLIAN LETTER ALI GALI THREE BALUDA
	0x1887: "a",           // MONGOLIAN LETTER ALI GALI A
	0x1888: "i",           // MONGOLIAN LETTER ALI GALI I
	0x1889: "k",           // MONGOLIAN LETTER ALI GALI KA
	0x188A: "ng",          // MONGOLIAN LETTER ALI GALI NGA
	0x188B: "c",           // MONGOLIAN LETTER ALI GALI CA
	0x188C: "tt",          // MONGOLIAN LETTER ALI GALI TTA
	0x188D: "tth",         // MONGOLIAN LETTER ALI GALI TTHA
	0x188E: "dd",          // MONGOLIAN LETTER ALI GALI DDA
	0x188F: "nn",          // MONGOLIAN LETTER ALI GALI NNA
	0x1890: "t",           // MONGOLIAN LETTER ALI GALI TA
	0x1891: "d",           // MONGOLIAN LETTER ALI GALI DA
	0x1892: "p",           // MONGOLIAN LETTER ALI GALI PA
	0x1893: "ph",          // MONGOLIAN LETTER ALI GALI PHA
	0x1894: "ss",          // MONGOLIAN LETTER ALI GALI SSA
	0x1895: "zh",          // MONGOLIAN LETTER ALI GALI ZHA
	0x1896: "z",           // MONGOLIAN LETTER ALI GALI ZA
	0x1897: "a",           // MONGOLIAN LETTER ALI GALI AH
	0x1898: "t",           // MONGOLIAN LETTER TODO ALI GALI TA
	0x1899: "zh",          // MONGOLIAN LETTER TODO ALI GALI ZHA
	0x189A: "gh",          // MONGOLIAN LETTER MANCHU ALI GALI GHA
	0x189B: "ng",          // MONGOLIAN LETTER MANCHU ALI GALI NGA
	0x189C: "c",           // MONGOLIAN LETTER MANCHU ALI GALI CA
	0x189D: "jh",          // MONGOLIAN LETTER MANCHU ALI GALI JHA
	0x189E: "tta",         // MONGOLIAN LETTER MANCHU ALI GALI TTA
	0x189F: "ddh",         // MONGOLIAN LETTER MANCHU ALI GALI DDHA
	0x18A0: "t",           // MONGOLIAN LETTER MANCHU ALI GALI TA
	0x18A1: "dh",          // MONGOLIAN LETTER MANCHU ALI GALI DHA
	0x18A2: "ss",          // MONGOLIAN LETTER MANCHU ALI GALI SSA
	0x18A3: "cy",          // MONGOLIAN LETTER MANCHU ALI GALI CYA
	0x18A4: "zh",          // MONGOLIAN LETTER MANCHU ALI GALI ZHA
	0x18A5: "z",           // MONGOLIAN LETTER MANCHU ALI GALI ZA
	0x18A6: "u",           // MONGOLIAN LETTER ALI GALI HALF U
	0x18A7: "y",           // MONGOLIAN LETTER ALI GALI HALF YA
	0x18A8: "bh",          // MONGOLIAN LETTER MANCHU ALI GALI BHA
	0x18A9: "'",           // MONGOLIAN LETTER ALI GALI DAGALGA
	0x1D5D: "b",           //
	0x1D66: "b",           //
	0x1D6C: "b",           //
	0x1D6D: "d",           //
	0x1D6E: "f",           //
	0x1D6F: "m",           //
	0x1D70: "n",           //
	0x1D71: "p",           //
	0x1D72: "r",           //
	0x1D73: "r",           //
	0x1D74: "s",           //
	0x1D75: "t",           //
	0x1D76: "z",           //
	0x1D77: "g",           //
	0x1D7D: "p",           //
	0x1D80: "b",           //
	0x1D81: "d",           //
	0x1D82: "f",           //
	0x1D83: "g",           //
	0x1D84: "k",           //
	0x1D85: "l",           //
	0x1D86: "m",           //
	0x1D87: "n",           //
	0x1D88: "p",           //
	0x1D89: "r",           //
	0x1D8A: "s",           //
	0x1D8C: "v",           //
	0x1D8D: "x",           //
	0x1D8E: "z",           //
	0x1E00: "A",           // LATIN CAPITAL LETTER A WITH RING BELOW
	0x1E01: "a",           // LATIN SMALL LETTER A WITH RING BELOW
	0x1E02: "B",           // LATIN CAPITAL LETTER B WITH DOT ABOVE
	0x1E03: "b",           // LATIN SMALL LETTER B WITH DOT ABOVE
	0x1E04: "B",           // LATIN CAPITAL LETTER B WITH DOT BELOW
	0x1E05: "b",           // LATIN SMALL LETTER B WITH DOT BELOW
	0x1E06: "B",           // LATIN CAPITAL LETTER B WITH LINE BELOW
	0x1E07: "b",           // LATIN SMALL LETTER B WITH LINE BELOW
	0x1E08: "C",           // LATIN CAPITAL LETTER C WITH CEDILLA AND ACUTE
	0x1E09: "c",           // LATIN SMALL LETTER C WITH CEDILLA AND ACUTE
	0x1E0A: "D",           // LATIN CAPITAL LETTER D WITH DOT ABOVE
	0x1E0B: "d",           // LATIN SMALL LETTER D WITH DOT ABOVE
	0x1E0C: "D",           // LATIN CAPITAL LETTER D WITH DOT BELOW
	0x1E0D: "d",           // LATIN SMALL LETTER D WITH DOT BELOW
	0x1E0E: "D",           // LATIN CAPITAL LETTER D WITH LINE BELOW
	0x1E0F: "d",           // LATIN SMALL LETTER D WITH LINE BELOW
	0x1E10: "D",           // LATIN CAPITAL LETTER D WITH CEDILLA
	0x1E11: "d",           // LATIN SMALL LETTER D WITH CEDILLA
	0x1E12: "D",           // LATIN CAPITAL LETTER D WITH CIRCUMFLEX BELOW
	0x1E13: "d",           // LATIN SMALL LETTER D WITH CIRCUMFLEX BELOW
	0x1E14: "E",           // LATIN CAPITAL LETTER E WITH MACRON AND GRAVE
	0x1E15: "e",           // LATIN SMALL LETTER E WITH MACRON AND GRAVE
	0x1E16: "E",           // LATIN CAPITAL LETTER E WITH MACRON AND ACUTE
	0x1E17: "e",           // LATIN SMALL LETTER E WITH MACRON AND ACUTE
	0x1E18: "E",           // LATIN CAPITAL LETTER E WITH CIRCUMFLEX BELOW
	0x1E19: "e",           // LATIN SMALL LETTER E WITH CIRCUMFLEX BELOW
	0x1E1A: "E",           // LATIN CAPITAL LETTER E WITH TILDE BELOW
	0x1E1B: "e",           // LATIN SMALL LETTER E WITH TILDE BELOW
	0x1E1C: "E",           // LATIN CAPITAL LETTER E WITH CEDILLA AND BREVE
	0x1E1D: "e",           // LATIN SMALL LETTER E WITH CEDILLA AND BREVE
	0x1E1E: "F",           // LATIN CAPITAL LETTER F WITH DOT ABOVE
	0x1E1F: "f",           // LATIN SMALL LETTER F WITH DOT ABOVE
	0x1E20: "G",           // LATIN CAPITAL LETTER G WITH MACRON
	0x1E21: "g",           // LATIN SMALL LETTER G WITH MACRON
	0x1E22: "H",           // LATIN CAPITAL LETTER H WITH DOT ABOVE
	0x1E23: "h",           // LATIN SMALL LETTER H WITH DOT ABOVE
	0x1E24: "H",           // LATIN CAPITAL LETTER H WITH DOT BELOW
	0x1E25: "h",           // LATIN SMALL LETTER H WITH DOT BELOW
	0x1E26: "H",           // LATIN CAPITAL LETTER H WITH DIAERESIS
	0x1E27: "h",           // LATIN SMALL LETTER H WITH DIAERESIS
	0x1E28: "H",           // LATIN CAPITAL LETTER H WITH CEDILLA
	0x1E29: "h",           // LATIN SMALL LETTER H WITH CEDILLA
	0x1E2A: "H",           // LATIN CAPITAL LETTER H WITH BREVE BELOW
	0x1E2B: "h",           // LATIN SMALL LETTER H WITH BREVE BELOW
	0x1E2C: "I",           // LATIN CAPITAL LETTER I WITH TILDE BELOW
	0x1E2D: "i",           // LATIN SMALL LETTER I WITH TILDE BELOW
	0x1E2E: "I",           // LATIN CAPITAL LETTER I WITH DIAERESIS AND ACUTE
	0x1E2F: "i",           // LATIN SMALL LETTER I WITH DIAERESIS AND ACUTE
	0x1E30: "K",           // LATIN CAPITAL LETTER K WITH ACUTE
	0x1E31: "k",           // LATIN SMALL LETTER K WITH ACUTE
	0x1E32: "K",           // LATIN CAPITAL LETTER K WITH DOT BELOW
	0x1E33: "k",           // LATIN SMALL LETTER K WITH DOT BELOW
	0x1E34: "K",           // LATIN CAPITAL LETTER K WITH LINE BELOW
	0x1E35: "k",           // LATIN SMALL LETTER K WITH LINE BELOW
	0x1E36: "L",           // LATIN CAPITAL LETTER L WITH DOT BELOW
	0x1E37: "l",           // LATIN SMALL LETTER L WITH DOT BELOW
	0x1E38: "L",           // LATIN CAPITAL LETTER L WITH DOT BELOW AND MACRON
	0x1E39: "l",           // LATIN SMALL LETTER L WITH DOT BELOW AND MACRON
	0x1E3A: "L",           // LATIN CAPITAL LETTER L WITH LINE BELOW
	0x1E3B: "l",           // LATIN SMALL LETTER L WITH LINE BELOW
	0x1E3C: "L",           // LATIN CAPITAL LETTER L WITH CIRCUMFLEX BELOW
	0x1E3D: "l",           // LATIN SMALL LETTER L WITH CIRCUMFLEX BELOW
	0x1E3E: "M",           // LATIN CAPITAL LETTER M WITH ACUTE
	0x1E3F: "m",           // LATIN SMALL LETTER M WITH ACUTE
	0x1E40: "M",           // LATIN CAPITAL LETTER M WITH DOT ABOVE
	0x1E41: "m",           // LATIN SMALL LETTER M WITH DOT ABOVE
	0x1E42: "M",           // LATIN CAPITAL LETTER M WITH DOT BELOW
	0x1E43: "m",           // LATIN SMALL LETTER M WITH DOT BELOW
	0x1E44: "N",           // LATIN CAPITAL LETTER N WITH DOT ABOVE
	0x1E45: "n",           // LATIN SMALL LETTER N WITH DOT ABOVE
	0x1E46: "N",           // LATIN CAPITAL LETTER N WITH DOT BELOW
	0x1E47: "n",           // LATIN SMALL LETTER N WITH DOT BELOW
	0x1E48: "N",           // LATIN CAPITAL LETTER N WITH LINE BELOW
	0x1E49: "n",           // LATIN SMALL LETTER N WITH LINE BELOW
	0x1E4A: "N",           // LATIN CAPITAL LETTER N WITH CIRCUMFLEX BELOW
	0x1E4B: "n",           // LATIN SMALL LETTER N WITH CIRCUMFLEX BELOW
	0x1E4C: "O",           // LATIN CAPITAL LETTER O WITH TILDE AND ACUTE
	0x1E4D: "o",           // LATIN SMALL LETTER O WITH TILDE AND ACUTE
	0x1E4E: "O",           // LATIN CAPITAL LETTER O WITH TILDE AND DIAERESIS
	0x1E4F: "o",           // LATIN SMALL LETTER O WITH TILDE AND DIAERESIS
	0x1E50: "O",           // LATIN CAPITAL LETTER O WITH MACRON AND GRAVE
	0x1E51: "o",           // LATIN SMALL LETTER O WITH MACRON AND GRAVE
	0x1E52: "O",           // LATIN CAPITAL LETTER O WITH MACRON AND ACUTE
	0x1E53: "o",           // LATIN SMALL LETTER O WITH MACRON AND ACUTE
	0x1E54: "P",           // LATIN CAPITAL LETTER P WITH ACUTE
	0x1E55: "p",           // LATIN SMALL LETTER P WITH ACUTE
	0x1E56: "P",           // LATIN CAPITAL LETTER P WITH DOT ABOVE
	0x1E57: "p",           // LATIN SMALL LETTER P WITH DOT ABOVE
	0x1E58: "R",           // LATIN CAPITAL LETTER R WITH DOT ABOVE
	0x1E59: "r",           // LATIN SMALL LETTER R WITH DOT ABOVE
	0x1E5A: "R",           // LATIN CAPITAL LETTER R WITH DOT BELOW
	0x1E5B: "r",           // LATIN SMALL LETTER R WITH DOT BELOW
	0x1E5C: "R",           // LATIN CAPITAL LETTER R WITH DOT BELOW AND MACRON
	0x1E5D: "r",           // LATIN SMALL LETTER R WITH DOT BELOW AND MACRON
	0x1E5E: "R",           // LATIN CAPITAL LETTER R WITH LINE BELOW
	0x1E5F: "r",           // LATIN SMALL LETTER R WITH LINE BELOW
	0x1E60: "S",           // LATIN CAPITAL LETTER S WITH DOT ABOVE
	0x1E61: "s",           // LATIN SMALL LETTER S WITH DOT ABOVE
	0x1E62: "S",           // LATIN CAPITAL LETTER S WITH DOT BELOW
	0x1E63: "s",           // LATIN SMALL LETTER S WITH DOT BELOW
	0x1E64: "S",           // LATIN CAPITAL LETTER S WITH ACUTE AND DOT ABOVE
	0x1E65: "s",           // LATIN SMALL LETTER S WITH ACUTE AND DOT ABOVE
	0x1E66: "S",           // LATIN CAPITAL LETTER S WITH CARON AND DOT ABOVE
	0x1E67: "s",           // LATIN SMALL LETTER S WITH CARON AND DOT ABOVE
	0x1E68: "S",           // LATIN CAPITAL LETTER S WITH DOT BELOW AND DOT ABOVE
	0x1E69: "s",           // LATIN SMALL LETTER S WITH DOT BELOW AND DOT ABOVE
	0x1E6A: "T",           // LATIN CAPITAL LETTER T WITH DOT ABOVE
	0x1E6B: "t",           // LATIN SMALL LETTER T WITH DOT ABOVE
	0x1E6C: "T",           // LATIN CAPITAL LETTER T WITH DOT BELOW
	0x1E6D: "t",           // LATIN SMALL LETTER T WITH DOT BELOW
	0x1E6E: "T",           // LATIN CAPITAL LETTER T WITH LINE BELOW
	0x1E6F: "t",           // LATIN SMALL LETTER T WITH LINE BELOW
	0x1E70: "T",           // LATIN CAPITAL LETTER T WITH CIRCUMFLEX BELOW
	0x1E71: "t",           // LATIN SMALL LETTER T WITH CIRCUMFLEX BELOW
	0x1E72: "U",           // LATIN CAPITAL LETTER U WITH DIAERESIS BELOW
	0x1E73: "u",           // LATIN SMALL LETTER U WITH DIAERESIS BELOW
	0x1E74: "U",           // LATIN CAPITAL LETTER U WITH TILDE BELOW
	0x1E75: "u",           // LATIN SMALL LETTER U WITH TILDE BELOW
	0x1E76: "U",           // LATIN CAPITAL LETTER U WITH CIRCUMFLEX BELOW
	0x1E77: "u",           // LATIN SMALL LETTER U WITH CIRCUMFLEX BELOW
	0x1E78: "U",           // LATIN CAPITAL LETTER U WITH TILDE AND ACUTE
	0x1E79: "u",           // LATIN SMALL LETTER U WITH TILDE AND ACUTE
	0x1E7A: "U",           // LATIN CAPITAL LETTER U WITH MACRON AND DIAERESIS
	0x1E7B: "u",           // LATIN SMALL LETTER U WITH MACRON AND DIAERESIS
	0x1E7C: "V",           // LATIN CAPITAL LETTER V WITH TILDE
	0x1E7D: "v",           // LATIN SMALL LETTER V WITH TILDE
	0x1E7E: "V",           // LATIN CAPITAL LETTER V WITH DOT BELOW
	0x1E7F: "v",           // LATIN SMALL LETTER V WITH DOT BELOW
	0x1E80: "W",           // LATIN CAPITAL LETTER W WITH GRAVE
	0x1E81: "w",           // LATIN SMALL LETTER W WITH GRAVE
	0x1E82: "W",           // LATIN CAPITAL LETTER W WITH ACUTE
	0x1E83: "w",           // LATIN SMALL LETTER W WITH ACUTE
	0x1E84: "W",           // LATIN CAPITAL LETTER W WITH DIAERESIS
	0x1E85: "w",           // LATIN SMALL LETTER W WITH DIAERESIS
	0x1E86: "W",           // LATIN CAPITAL LETTER W WITH DOT ABOVE
	0x1E87: "w",           // LATIN SMALL LETTER W WITH DOT ABOVE
	0x1E88: "W",           // LATIN CAPITAL LETTER W WITH DOT BELOW
	0x1E89: "w",           // LATIN SMALL LETTER W WITH DOT BELOW
	0x1E8A: "X",           // LATIN CAPITAL LETTER X WITH DOT ABOVE
	0x1E8B: "x",           // LATIN SMALL LETTER X WITH DOT ABOVE
	0x1E8C: "X",           // LATIN CAPITAL LETTER X WITH DIAERESIS
	0x1E8D: "x",           // LATIN SMALL LETTER X WITH DIAERESIS
	0x1E8E: "Y",           // LATIN CAPITAL LETTER Y WITH DOT ABOVE
	0x1E8F: "y",           // LATIN SMALL LETTER Y WITH DOT ABOVE
	0x1E90: "Z",           // LATIN CAPITAL LETTER Z WITH CIRCUMFLEX
	0x1E91: "z",           // LATIN SMALL LETTER Z WITH CIRCUMFLEX
	0x1E92: "Z",           // LATIN CAPITAL LETTER Z WITH DOT BELOW
	0x1E93: "z",           // LATIN SMALL LETTER Z WITH DOT BELOW
	0x1E94: "Z",           // LATIN CAPITAL LETTER Z WITH LINE BELOW
	0x1E95: "z",           // LATIN SMALL LETTER Z WITH LINE BELOW
	0x1E96: "h",           // LATIN SMALL LETTER H WITH LINE BELOW
	0x1E97: "t",           // LATIN SMALL LETTER T WITH DIAERESIS
	0x1E98: "w",           // LATIN SMALL LETTER W WITH RING ABOVE
	0x1E99: "y",           // LATIN SMALL LETTER Y WITH RING ABOVE
	0x1E9A: "a",           // LATIN SMALL LETTER A WITH RIGHT HALF RING
	0x1E9B: "S",           // LATIN SMALL LETTER LONG S WITH DOT ABOVE
	0x1E9E: "SS",          //
	0x1EA0: "A",           // LATIN CAPITAL LETTER A WITH DOT BELOW
	0x1EA1: "a",           // LATIN SMALL LETTER A WITH DOT BELOW
	0x1EA2: "A",           // LATIN CAPITAL LETTER A WITH HOOK ABOVE
	0x1EA3: "a",           // LATIN SMALL LETTER A WITH HOOK ABOVE
	0x1EA4: "A",           // LATIN CAPITAL LETTER A WITH CIRCUMFLEX AND ACUTE
	0x1EA5: "a",           // LATIN SMALL LETTER A WITH CIRCUMFLEX AND ACUTE
	0x1EA6: "A",           // LATIN CAPITAL LETTER A WITH CIRCUMFLEX AND GRAVE
	0x1EA7: "a",           // LATIN SMALL LETTER A WITH CIRCUMFLEX AND GRAVE
	0x1EA8: "A",           // LATIN CAPITAL LETTER A WITH CIRCUMFLEX AND HOOK ABOVE
	0x1EA9: "a",           // LATIN SMALL LETTER A WITH CIRCUMFLEX AND HOOK ABOVE
	0x1EAA: "A",           // LATIN CAPITAL LETTER A WITH CIRCUMFLEX AND TILDE
	0x1EAB: "a",           // LATIN SMALL LETTER A WITH CIRCUMFLEX AND TILDE
	0x1EAC: "A",           // LATIN CAPITAL LETTER A WITH CIRCUMFLEX AND DOT BELOW
	0x1EAD: "a",           // LATIN SMALL LETTER A WITH CIRCUMFLEX AND DOT BELOW
	0x1EAE: "A",           // LATIN CAPITAL LETTER A WITH BREVE AND ACUTE
	0x1EAF: "a",           // LATIN SMALL LETTER A WITH BREVE AND ACUTE
	0x1EB0: "A",           // LATIN CAPITAL LETTER A WITH BREVE AND GRAVE
	0x1EB1: "a",           // LATIN SMALL LETTER A WITH BREVE AND GRAVE
	0x1EB2: "A",           // LATIN CAPITAL LETTER A WITH BREVE AND HOOK ABOVE
	0x1EB3: "a",           // LATIN SMALL LETTER A WITH BREVE AND HOOK ABOVE
	0x1EB4: "A",           // LATIN CAPITAL LETTER A WITH BREVE AND TILDE
	0x1EB5: "a",           // LATIN SMALL LETTER A WITH BREVE AND TILDE
	0x1EB6: "A",           // LATIN CAPITAL LETTER A WITH BREVE AND DOT BELOW
	0x1EB7: "a",           // LATIN SMALL LETTER A WITH BREVE AND DOT BELOW
	0x1EB8: "E",           // LATIN CAPITAL LETTER E WITH DOT BELOW
	0x1EB9: "e",           // LATIN SMALL LETTER E WITH DOT BELOW
	0x1EBA: "E",           // LATIN CAPITAL LETTER E WITH HOOK ABOVE
	0x1EBB: "e",           // LATIN SMALL LETTER E WITH HOOK ABOVE
	0x1EBC: "E",           // LATIN CAPITAL LETTER E WITH TILDE
	0x1EBD: "e",           // LATIN SMALL LETTER E WITH TILDE
	0x1EBE: "E",           // LATIN CAPITAL LETTER E WITH CIRCUMFLEX AND ACUTE
	0x1EBF: "e",           // LATIN SMALL LETTER E WITH CIRCUMFLEX AND ACUTE
	0x1EC0: "E",           // LATIN CAPITAL LETTER E WITH CIRCUMFLEX AND GRAVE
	0x1EC1: "e",           // LATIN SMALL LETTER E WITH CIRCUMFLEX AND GRAVE
	0x1EC2: "E",           // LATIN CAPITAL LETTER E WITH CIRCUMFLEX AND HOOK ABOVE
	0x1EC3: "e",           // LATIN SMALL LETTER E WITH CIRCUMFLEX AND HOOK ABOVE
	0x1EC4: "E",           // LATIN CAPITAL LETTER E WITH CIRCUMFLEX AND TILDE
	0x1EC5: "e",           // LATIN SMALL LETTER E WITH CIRCUMFLEX AND TILDE
	0x1EC6: "E",           // LATIN CAPITAL LETTER E WITH CIRCUMFLEX AND DOT BELOW
	0x1EC7: "e",           // LATIN SMALL LETTER E WITH CIRCUMFLEX AND DOT BELOW
	0x1EC8: "I",           // LATIN CAPITAL LETTER I WITH HOOK ABOVE
	0x1EC9: "i",           // LATIN SMALL LETTER I WITH HOOK ABOVE
	0x1ECA: "I",           // LATIN CAPITAL LETTER I WITH DOT BELOW
	0x1ECB: "i",           // LATIN SMALL LETTER I WITH DOT BELOW
	0x1ECC: "O",           // LATIN CAPITAL LETTER O WITH DOT BELOW
	0x1ECD: "o",           // LATIN SMALL LETTER O WITH DOT BELOW
	0x1ECE: "O",           // LATIN CAPITAL LETTER O WITH HOOK ABOVE
	0x1ECF: "o",           // LATIN SMALL LETTER O WITH HOOK ABOVE
	0x1ED0: "O",           // LATIN CAPITAL LETTER O WITH CIRCUMFLEX AND ACUTE
	0x1ED1: "o",           // LATIN SMALL LETTER O WITH CIRCUMFLEX AND ACUTE
	0x1ED2: "O",           // LATIN CAPITAL LETTER O WITH CIRCUMFLEX AND GRAVE
	0x1ED3: "o",           // LATIN SMALL LETTER O WITH CIRCUMFLEX AND GRAVE
	0x1ED4: "O",           // LATIN CAPITAL LETTER O WITH CIRCUMFLEX AND HOOK ABOVE
	0x1ED5: "o",           // LATIN SMALL LETTER O WITH CIRCUMFLEX AND HOOK ABOVE
	0x1ED6: "O",           // LATIN CAPITAL LETTER O WITH CIRCUMFLEX AND TILDE
	0x1ED7: "o",           // LATIN SMALL LETTER O WITH CIRCUMFLEX AND TILDE
	0x1ED8: "O",           // LATIN CAPITAL LETTER O WITH CIRCUMFLEX AND DOT BELOW
	0x1ED9: "o",           // LATIN SMALL LETTER O WITH CIRCUMFLEX AND DOT BELOW
	0x1EDA: "O",           // LATIN CAPITAL LETTER O WITH HORN AND ACUTE
	0x1EDB: "o",           // LATIN SMALL LETTER O WITH HORN AND ACUTE
	0x1EDC: "O",           // LATIN CAPITAL LETTER O WITH HORN AND GRAVE
	0x1EDD: "o",           // LATIN SMALL LETTER O WITH HORN AND GRAVE
	0x1EDE: "O",           // LATIN CAPITAL LETTER O WITH HORN AND HOOK ABOVE
	0x1EDF: "o",           // LATIN SMALL LETTER O WITH HORN AND HOOK ABOVE
	0x1EE0: "O",           // LATIN CAPITAL LETTER O WITH HORN AND TILDE
	0x1EE1: "o",           // LATIN SMALL LETTER O WITH HORN AND TILDE
	0x1EE2: "O",           // LATIN CAPITAL LETTER O WITH HORN AND DOT BELOW
	0x1EE3: "o",           // LATIN SMALL LETTER O WITH HORN AND DOT BELOW
	0x1EE4: "U",           // LATIN CAPITAL LETTER U WITH DOT BELOW
	0x1EE5: "u",           // LATIN SMALL LETTER U WITH DOT BELOW
	0x1EE6: "U",           // LATIN CAPITAL LETTER U WITH HOOK ABOVE
	0x1EE7: "u",           // LATIN SMALL LETTER U WITH HOOK ABOVE
	0x1EE8: "U",           // LATIN CAPITAL LETTER U WITH HORN AND ACUTE
	0x1EE9: "u",           // LATIN SMALL LETTER U WITH HORN AND ACUTE
	0x1EEA: "U",           // LATIN CAPITAL LETTER U WITH HORN AND GRAVE
	0x1EEB: "u",           // LATIN SMALL LETTER U WITH HORN AND GRAVE
	0x1EEC: "U",           // LATIN CAPITAL LETTER U WITH HORN AND HOOK ABOVE
	0x1EED: "u",           // LATIN SMALL LETTER U WITH HORN AND HOOK ABOVE
	0x1EEE: "U",           // LATIN CAPITAL LETTER U WITH HORN AND TILDE
	0x1EEF: "u",           // LATIN SMALL LETTER U WITH HORN AND TILDE
	0x1EF0: "U",           // LATIN CAPITAL LETTER U WITH HORN AND DOT BELOW
	0x1EF1: "u",           // LATIN SMALL LETTER U WITH HORN AND DOT BELOW
	0x1EF2: "Y",           // LATIN CAPITAL LETTER Y WITH GRAVE
	0x1EF3: "y",           // LATIN SMALL LETTER Y WITH GRAVE
	0x1EF4: "Y",           // LATIN CAPITAL LETTER Y WITH DOT BELOW
	0x1EF5: "y",           // LATIN SMALL LETTER Y WITH DOT BELOW
	0x1EF6: "Y",           // LATIN CAPITAL LETTER Y WITH HOOK ABOVE
	0x1EF7: "y",           // LATIN SMALL LETTER Y WITH HOOK ABOVE
	0x1EF8: "Y",           // LATIN CAPITAL LETTER Y WITH TILDE
	0x1EF9: "y",           // LATIN SMALL LETTER Y WITH TILDE
	0x1F00: "a",           // GREEK SMALL LETTER ALPHA WITH PSILI
	0x1F01: "a",           // GREEK SMALL LETTER ALPHA WITH DASIA
	0x1F02: "a",           // GREEK SMALL LETTER ALPHA WITH PSILI AND VARIA
	0x1F03: "a",           // GREEK SMALL LETTER ALPHA WITH DASIA AND VARIA
	0x1F04: "a",           // GREEK SMALL LETTER ALPHA WITH PSILI AND OXIA
	0x1F05: "a",           // GREEK SMALL LETTER ALPHA WITH DASIA AND OXIA
	0x1F06: "a",           // GREEK SMALL LETTER ALPHA WITH PSILI AND PERISPOMENI
	0x1F07: "a",           // GREEK SMALL LETTER ALPHA WITH DASIA AND PERISPOMENI
	0x1F08: "A",           // GREEK CAPITAL LETTER ALPHA WITH PSILI
	0x1F09: "A",           // GREEK CAPITAL LETTER ALPHA WITH DASIA
	0x1F0A: "A",           // GREEK CAPITAL LETTER ALPHA WITH PSILI AND VARIA
	0x1F0B: "A",           // GREEK CAPITAL LETTER ALPHA WITH DASIA AND VARIA
	0x1F0C: "A",           // GREEK CAPITAL LETTER ALPHA WITH PSILI AND OXIA
	0x1F0D: "A",           // GREEK CAPITAL LETTER ALPHA WITH DASIA AND OXIA
	0x1F0E: "A",           // GREEK CAPITAL LETTER ALPHA WITH PSILI AND PERISPOMENI
	0x1F0F: "A",           // GREEK CAPITAL LETTER ALPHA WITH DASIA AND PERISPOMENI
	0x1F10: "e",           // GREEK SMALL LETTER EPSILON WITH PSILI
	0x1F11: "e",           // GREEK SMALL LETTER EPSILON WITH DASIA
	0x1F12: "e",           // GREEK SMALL LETTER EPSILON WITH PSILI AND VARIA
	0x1F13: "e",           // GREEK SMALL LETTER EPSILON WITH DASIA AND VARIA
	0x1F14: "e",           // GREEK SMALL LETTER EPSILON WITH PSILI AND OXIA
	0x1F15: "e",           // GREEK SMALL LETTER EPSILON WITH DASIA AND OXIA
	0x1F18: "E",           // GREEK CAPITAL LETTER EPSILON WITH PSILI
	0x1F19: "E",           // GREEK CAPITAL LETTER EPSILON WITH DASIA
	0x1F1A: "E",           // GREEK CAPITAL LETTER EPSILON WITH PSILI AND VARIA
	0x1F1B: "E",           // GREEK CAPITAL LETTER EPSILON WITH DASIA AND VARIA
	0x1F1C: "E",           // GREEK CAPITAL LETTER EPSILON WITH PSILI AND OXIA
	0x1F1D: "E",           // GREEK CAPITAL LETTER EPSILON WITH DASIA AND OXIA
	0x1F20: "e",           // GREEK SMALL LETTER ETA WITH PSILI
	0x1F21: "e",           // GREEK SMALL LETTER ETA WITH DASIA
	0x1F22: "e",           // GREEK SMALL LETTER ETA WITH PSILI AND VARIA
	0x1F23: "e",           // GREEK SMALL LETTER ETA WITH DASIA AND VARIA
	0x1F24: "e",           // GREEK SMALL LETTER ETA WITH PSILI AND OXIA
	0x1F25: "e",           // GREEK SMALL LETTER ETA WITH DASIA AND OXIA
	0x1F26: "e",           // GREEK SMALL LETTER ETA WITH PSILI AND PERISPOMENI
	0x1F27: "e",           // GREEK SMALL LETTER ETA WITH DASIA AND PERISPOMENI
	0x1F28: "E",           // GREEK CAPITAL LETTER ETA WITH PSILI
	0x1F29: "E",           // GREEK CAPITAL LETTER ETA WITH DASIA
	0x1F2A: "E",           // GREEK CAPITAL LETTER ETA WITH PSILI AND VARIA
	0x1F2B: "E",           // GREEK CAPITAL LETTER ETA WITH DASIA AND VARIA
	0x1F2C: "E",           // GREEK CAPITAL LETTER ETA WITH PSILI AND OXIA
	0x1F2D: "E",           // GREEK CAPITAL LETTER ETA WITH DASIA AND OXIA
	0x1F2E: "E",           // GREEK CAPITAL LETTER ETA WITH PSILI AND PERISPOMENI
	0x1F2F: "E",           // GREEK CAPITAL LETTER ETA WITH DASIA AND PERISPOMENI
	0x1F30: "i",           // GREEK SMALL LETTER IOTA WITH PSILI
	0x1F31: "i",           // GREEK SMALL LETTER IOTA WITH DASIA
	0x1F32: "i",           // GREEK SMALL LETTER IOTA WITH PSILI AND VARIA
	0x1F33: "i",           // GREEK SMALL LETTER IOTA WITH DASIA AND VARIA
	0x1F34: "i",           // GREEK SMALL LETTER IOTA WITH PSILI AND OXIA
	0x1F35: "i",           // GREEK SMALL LETTER IOTA WITH DASIA AND OXIA
	0x1F36: "i",           // GREEK SMALL LETTER IOTA WITH PSILI AND PERISPOMENI
	0x1F37: "i",           // GREEK SMALL LETTER IOTA WITH DASIA AND PERISPOMENI
	0x1F38: "I",           // GREEK CAPITAL LETTER IOTA WITH PSILI
	0x1F39: "I",           // GREEK CAPITAL LETTER IOTA WITH DASIA
	0x1F3A: "I",           // GREEK CAPITAL LETTER IOTA WITH PSILI AND VARIA
	0x1F3B: "I",           // GREEK CAPITAL LETTER IOTA WITH DASIA AND VARIA
	0x1F3C: "I",           // GREEK CAPITAL LETTER IOTA WITH PSILI AND OXIA
	0x1F3D: "I",           // GREEK CAPITAL LETTER IOTA WITH DASIA AND OXIA
	0x1F3E: "I",           // GREEK CAPITAL LETTER IOTA WITH PSILI AND PERISPOMENI
	0x1F3F: "I",           // GREEK CAPITAL LETTER IOTA WITH DASIA AND PERISPOMENI
	0x1F40: "o",           // GREEK SMALL LETTER OMICRON WITH PSILI
	0x1F41: "o",           // GREEK SMALL LETTER OMICRON WITH DASIA
	0x1F42: "o",           // GREEK SMALL LETTER OMICRON WITH PSILI AND VARIA
	0x1F43: "o",           // GREEK SMALL LETTER OMICRON WITH DASIA AND VARIA
	0x1F44: "o",           // GREEK SMALL LETTER OMICRON WITH PSILI AND OXIA
	0x1F45: "o",           // GREEK SMALL LETTER OMICRON WITH DASIA AND OXIA
	0x1F48: "O",           // GREEK CAPITAL LETTER OMICRON WITH PSILI
	0x1F49: "O",           // GREEK CAPITAL LETTER OMICRON WITH DASIA
	0x1F4A: "O",           // GREEK CAPITAL LETTER OMICRON WITH PSILI AND VARIA
	0x1F4B: "O",           // GREEK CAPITAL LETTER OMICRON WITH DASIA AND VARIA
	0x1F4C: "O",           // GREEK CAPITAL LETTER OMICRON WITH PSILI AND OXIA
	0x1F4D: "O",           // GREEK CAPITAL LETTER OMICRON WITH DASIA AND OXIA
	0x1F50: "u",           // GREEK SMALL LETTER UPSILON WITH PSILI
	0x1F51: "u",           // GREEK SMALL LETTER UPSILON WITH DASIA
	0x1F52: "u",           // GREEK SMALL LETTER UPSILON WITH PSILI AND VARIA
	0x1F53: "u",           // GREEK SMALL LETTER UPSILON WITH DASIA AND VARIA
	0x1F54: "u",           // GREEK SMALL LETTER UPSILON WITH PSILI AND OXIA
	0x1F55: "u",           // GREEK SMALL LETTER UPSILON WITH DASIA AND OXIA
	0x1F56: "u",           // GREEK SMALL LETTER UPSILON WITH PSILI AND PERISPOMENI
	0x1F57: "u",           // GREEK SMALL LETTER UPSILON WITH DASIA AND PERISPOMENI
	0x1F59: "U",           // GREEK CAPITAL LETTER UPSILON WITH DASIA
	0x1F5B: "U",           // GREEK CAPITAL LETTER UPSILON WITH DASIA AND VARIA
	0x1F5D: "U",           // GREEK CAPITAL LETTER UPSILON WITH DASIA AND OXIA
	0x1F5F: "U",           // GREEK CAPITAL LETTER UPSILON WITH DASIA AND PERISPOMENI
	0x1F60: "o",           // GREEK SMALL LETTER OMEGA WITH PSILI
	0x1F61: "o",           // GREEK SMALL LETTER OMEGA WITH DASIA
	0x1F62: "o",           // GREEK SMALL LETTER OMEGA WITH PSILI AND VARIA
	0x1F63: "o",           // GREEK SMALL LETTER OMEGA WITH DASIA AND VARIA
	0x1F64: "o",           // GREEK SMALL LETTER OMEGA WITH PSILI AND OXIA
	0x1F65: "o",           // GREEK SMALL LETTER OMEGA WITH DASIA AND OXIA
	0x1F66: "o",           // GREEK SMALL LETTER OMEGA WITH PSILI AND PERISPOMENI
	0x1F67: "o",           // GREEK SMALL LETTER OMEGA WITH DASIA AND PERISPOMENI
	0x1F68: "O",           // GREEK CAPITAL LETTER OMEGA WITH PSILI
	0x1F69: "O",           // GREEK CAPITAL LETTER OMEGA WITH DASIA
	0x1F6A: "O",           // GREEK CAPITAL LETTER OMEGA WITH PSILI AND VARIA
	0x1F6B: "O",           // GREEK CAPITAL LETTER OMEGA WITH DASIA AND VARIA
	0x1F6C: "O",           // GREEK CAPITAL LETTER OMEGA WITH PSILI AND OXIA
	0x1F6D: "O",           // GREEK CAPITAL LETTER OMEGA WITH DASIA AND OXIA
	0x1F6E: "O",           // GREEK CAPITAL LETTER OMEGA WITH PSILI AND PERISPOMENI
	0x1F6F: "O",           // GREEK CAPITAL LETTER OMEGA WITH DASIA AND PERISPOMENI
	0x1F70: "a",           // GREEK SMALL LETTER ALPHA WITH VARIA
	0x1F71: "a",           // GREEK SMALL LETTER ALPHA WITH OXIA
	0x1F72: "e",           // GREEK SMALL LETTER EPSILON WITH VARIA
	0x1F73: "e",           // GREEK SMALL LETTER EPSILON WITH OXIA
	0x1F74: "e",           // GREEK SMALL LETTER ETA WITH VARIA
	0x1F75: "e",           // GREEK SMALL LETTER ETA WITH OXIA
	0x1F76: "i",           // GREEK SMALL LETTER IOTA WITH VARIA
	0x1F77: "i",           // GREEK SMALL LETTER IOTA WITH OXIA
	0x1F78: "o",           // GREEK SMALL LETTER OMICRON WITH VARIA
	0x1F79: "o",           // GREEK SMALL LETTER OMICRON WITH OXIA
	0x1F7A: "u",           // GREEK SMALL LETTER UPSILON WITH VARIA
	0x1F7B: "u",           // GREEK SMALL LETTER UPSILON WITH OXIA
	0x1F7C: "o",           // GREEK SMALL LETTER OMEGA WITH VARIA
	0x1F7D: "o",           // GREEK SMALL LETTER OMEGA WITH OXIA
	0x1F80: "a",           // GREEK SMALL LETTER ALPHA WITH PSILI AND YPOGEGRAMMENI
	0x1F81: "a",           // GREEK SMALL LETTER ALPHA WITH DASIA AND YPOGEGRAMMENI
	0x1F82: "a",           // GREEK SMALL LETTER ALPHA WITH PSILI AND VARIA AND YPOGEGRAMMENI
	0x1F83: "a",           // GREEK SMALL LETTER ALPHA WITH DASIA AND VARIA AND YPOGEGRAMMENI
	0x1F84: "a",           // GREEK SMALL LETTER ALPHA WITH PSILI AND OXIA AND YPOGEGRAMMENI
	0x1F85: "a",           // GREEK SMALL LETTER ALPHA WITH DASIA AND OXIA AND YPOGEGRAMMENI
	0x1F86: "a",           // GREEK SMALL LETTER ALPHA WITH PSILI AND PERISPOMENI AND YPOGEGRAMMENI
	0x1F87: "a",           // GREEK SMALL LETTER ALPHA WITH DASIA AND PERISPOMENI AND YPOGEGRAMMENI
	0x1F88: "A",           // GREEK CAPITAL LETTER ALPHA WITH PSILI AND PROSGEGRAMMENI
	0x1F89: "A",           // GREEK CAPITAL LETTER ALPHA WITH DASIA AND PROSGEGRAMMENI
	0x1F8A: "A",           // GREEK CAPITAL LETTER ALPHA WITH PSILI AND VARIA AND PROSGEGRAMMENI
	0x1F8B: "A",           // GREEK CAPITAL LETTER ALPHA WITH DASIA AND VARIA AND PROSGEGRAMMENI
	0x1F8C: "A",           // GREEK CAPITAL LETTER ALPHA WITH PSILI AND OXIA AND PROSGEGRAMMENI
	0x1F8D: "A",           // GREEK CAPITAL LETTER ALPHA WITH DASIA AND OXIA AND PROSGEGRAMMENI
	0x1F8E: "A",           // GREEK CAPITAL LETTER ALPHA WITH PSILI AND PERISPOMENI AND PROSGEGRAMM
	0x1F8F: "A",           // GREEK CAPITAL LETTER ALPHA WITH DASIA AND PERISPOMENI AND PROSGEGRAMM
	0x1F90: "e",           // GREEK SMALL LETTER ETA WITH PSILI AND YPOGEGRAMMENI
	0x1F91: "e",           // GREEK SMALL LETTER ETA WITH DASIA AND YPOGEGRAMMENI
	0x1F92: "e",           // GREEK SMALL LETTER ETA WITH PSILI AND VARIA AND YPOGEGRAMMENI
	0x1F93: "e",           // GREEK SMALL LETTER ETA WITH DASIA AND VARIA AND YPOGEGRAMMENI
	0x1F94: "e",           // GREEK SMALL LETTER ETA WITH PSILI AND OXIA AND YPOGEGRAMMENI
	0x1F95: "e",           // GREEK SMALL LETTER ETA WITH DASIA AND OXIA AND YPOGEGRAMMENI
	0x1F96: "e",           // GREEK SMALL LETTER ETA WITH PSILI AND PERISPOMENI AND YPOGEGRAMMENI
	0x1F97: "e",           // GREEK SMALL LETTER ETA WITH DASIA AND PERISPOMENI AND YPOGEGRAMMENI
	0x1F98: "E",           // GREEK CAPITAL LETTER ETA WITH PSILI AND PROSGEGRAMMENI
	0x1F99: "E",           // GREEK CAPITAL LETTER ETA WITH DASIA AND PROSGEGRAMMENI
	0x1F9A: "E",           // GREEK CAPITAL LETTER ETA WITH PSILI AND VARIA AND PROSGEGRAMMENI
	0x1F9B: "E",           // GREEK CAPITAL LETTER ETA WITH DASIA AND VARIA AND PROSGEGRAMMENI
	0x1F9C: "E",           // GREEK CAPITAL LETTER ETA WITH PSILI AND OXIA AND PROSGEGRAMMENI
	0x1F9D: "E",           // GREEK CAPITAL LETTER ETA WITH DASIA AND OXIA AND PROSGEGRAMMENI
	0x1F9E: "E",           // GREEK CAPITAL LETTER ETA WITH PSILI AND PERISPOMENI AND PROSGEGRAMMEN
	0x1F9F: "E",           // GREEK CAPITAL LETTER ETA WITH DASIA AND PERISPOMENI AND PROSGEGRAMMEN
	0x1FA0: "o",           // GREEK SMALL LETTER OMEGA WITH PSILI AND YPOGEGRAMMENI
	0x1FA1: "o",           // GREEK SMALL LETTER OMEGA WITH DASIA AND YPOGEGRAMMENI
	0x1FA2: "o",           // GREEK SMALL LETTER OMEGA WITH PSILI AND VARIA AND YPOGEGRAMMENI
	0x1FA3: "o",           // GREEK SMALL LETTER OMEGA WITH DASIA AND VARIA AND YPOGEGRAMMENI
	0x1FA4: "o",           // GREEK SMALL LETTER OMEGA WITH PSILI AND OXIA AND YPOGEGRAMMENI
	0x1FA5: "o",           // GREEK SMALL LETTER OMEGA WITH DASIA AND OXIA AND YPOGEGRAMMENI
	0x1FA6: "o",           // GREEK SMALL LETTER OMEGA WITH PSILI AND PERISPOMENI AND YPOGEGRAMMENI
	0x1FA7: "o",           // GREEK SMALL LETTER OMEGA WITH DASIA AND PERISPOMENI AND YPOGEGRAMMENI
	0x1FA8: "O",           // GREEK CAPITAL LETTER OMEGA WITH PSILI AND PROSGEGRAMMENI
	0x1FA9: "O",           // GREEK CAPITAL LETTER OMEGA WITH DASIA AND PROSGEGRAMMENI
	0x1FAA: "O",           // GREEK CAPITAL LETTER OMEGA WITH PSILI AND VARIA AND PROSGEGRAMMENI
	0x1FAB: "O",           // GREEK CAPITAL LETTER OMEGA WITH DASIA AND VARIA AND PROSGEGRAMMENI
	0x1FAC: "O",           // GREEK CAPITAL LETTER OMEGA WITH PSILI AND OXIA AND PROSGEGRAMMENI
	0x1FAD: "O",           // GREEK CAPITAL LETTER OMEGA WITH DASIA AND OXIA AND PROSGEGRAMMENI
	0x1FAE: "O",           // GREEK CAPITAL LETTER OMEGA WITH PSILI AND PERISPOMENI AND PROSGEGRAMM
	0x1FAF: "O",           // GREEK CAPITAL LETTER OMEGA WITH DASIA AND PERISPOMENI AND PROSGEGRAMM
	0x1FB0: "a",           // GREEK SMALL LETTER ALPHA WITH VRACHY
	0x1FB1: "a",           // GREEK SMALL LETTER ALPHA WITH MACRON
	0x1FB2: "a",           // GREEK SMALL LETTER ALPHA WITH VARIA AND YPOGEGRAMMENI
	0x1FB3: "a",           // GREEK SMALL LETTER ALPHA WITH YPOGEGRAMMENI
	0x1FB4: "a",           // GREEK SMALL LETTER ALPHA WITH OXIA AND YPOGEGRAMMENI
	0x1FB6: "a",           // GREEK SMALL LETTER ALPHA WITH PERISPOMENI
	0x1FB7: "a",           // GREEK SMALL LETTER ALPHA WITH PERISPOMENI AND YPOGEGRAMMENI
	0x1FB8: "A",           // GREEK CAPITAL LETTER ALPHA WITH VRACHY
	0x1FB9: "A",           // GREEK CAPITAL LETTER ALPHA WITH MACRON
	0x1FBA: "A",           // GREEK CAPITAL LETTER ALPHA WITH VARIA
	0x1FBB: "A",           // GREEK CAPITAL LETTER ALPHA WITH OXIA
	0x1FBC: "A",           // GREEK CAPITAL LETTER ALPHA WITH PROSGEGRAMMENI
	0x1FBD: "'",           // GREEK KORONIS
	0x1FBE: "i",           // GREEK PROSGEGRAMMENI
	0x1FBF: "'",           // GREEK PSILI
	0x1FC0: "~",           // GREEK PERISPOMENI
	0x1FC1: "\"~",         // GREEK DIALYTIKA AND PERISPOMENI
	0x1FC2: "e",           // GREEK SMALL LETTER ETA WITH VARIA AND YPOGEGRAMMENI
	0x1FC3: "e",           // GREEK SMALL LETTER ETA WITH YPOGEGRAMMENI
	0x1FC4: "e",           // GREEK SMALL LETTER ETA WITH OXIA AND YPOGEGRAMMENI
	0x1FC6: "e",           // GREEK SMALL LETTER ETA WITH PERISPOMENI
	0x1FC7: "e",           // GREEK SMALL LETTER ETA WITH PERISPOMENI AND YPOGEGRAMMENI
	0x1FC8: "E",           // GREEK CAPITAL LETTER EPSILON WITH VARIA
	0x1FC9: "E",           // GREEK CAPITAL LETTER EPSILON WITH OXIA
	0x1FCA: "E",           // GREEK CAPITAL LETTER ETA WITH VARIA
	0x1FCB: "E",           // GREEK CAPITAL LETTER ETA WITH OXIA
	0x1FCC: "E",           // GREEK CAPITAL LETTER ETA WITH PROSGEGRAMMENI
	0x1FCD: "'`",          // GREEK PSILI AND VARIA
	0x1FCE: "''",          // GREEK PSILI AND OXIA
	0x1FCF: "'~",          // GREEK PSILI AND PERISPOMENI
	0x1FD0: "i",           // GREEK SMALL LETTER IOTA WITH VRACHY
	0x1FD1: "i",           // GREEK SMALL LETTER IOTA WITH MACRON
	0x1FD2: "i",           // GREEK SMALL LETTER IOTA WITH DIALYTIKA AND VARIA
	0x1FD3: "i",           // GREEK SMALL LETTER IOTA WITH DIALYTIKA AND OXIA
	0x1FD6: "i",           // GREEK SMALL LETTER IOTA WITH PERISPOMENI
	0x1FD7: "i",           // GREEK SMALL LETTER IOTA WITH DIALYTIKA AND PERISPOMENI
	0x1FD8: "I",           // GREEK CAPITAL LETTER IOTA WITH VRACHY
	0x1FD9: "I",           // GREEK CAPITAL LETTER IOTA WITH MACRON
	0x1FDA: "I",           // GREEK CAPITAL LETTER IOTA WITH VARIA
	0x1FDB: "I",           // GREEK CAPITAL LETTER IOTA WITH OXIA
	0x1FDD: "`'",          // GREEK DASIA AND VARIA
	0x1FDE: "`'",          // GREEK DASIA AND OXIA
	0x1FDF: "`~",          // GREEK DASIA AND PERISPOMENI
	0x1FE0: "u",           // GREEK SMALL LETTER UPSILON WITH VRACHY
	0x1FE1: "u",           // GREEK SMALL LETTER UPSILON WITH MACRON
	0x1FE2: "u",           // GREEK SMALL LETTER UPSILON WITH DIALYTIKA AND VARIA
	0x1FE3: "u",           // GREEK SMALL LETTER UPSILON WITH DIALYTIKA AND OXIA
	0x1FE4: "R",           // GREEK SMALL LETTER RHO WITH PSILI
	0x1FE5: "R",           // GREEK SMALL LETTER RHO WITH DASIA
	0x1FE6: "u",           // GREEK SMALL LETTER UPSILON WITH PERISPOMENI
	0x1FE7: "u",           // GREEK SMALL LETTER UPSILON WITH DIALYTIKA AND PERISPOMENI
	0x1FE8: "U",           // GREEK CAPITAL LETTER UPSILON WITH VRACHY
	0x1FE9: "U",           // GREEK CAPITAL LETTER UPSILON WITH MACRON
	0x1FEA: "U",           // GREEK CAPITAL LETTER UPSILON WITH VARIA
	0x1FEB: "U",           // GREEK CAPITAL LETTER UPSILON WITH OXIA
	0x1FEC: "R",           // GREEK CAPITAL LETTER RHO WITH DASIA
	0x1FED: "\"`",         // GREEK DIALYTIKA AND VARIA
	0x1FEE: "\"'",         // GREEK DIALYTIKA AND OXIA
	0x1FEF: "`",           // GREEK VARIA
	0x1FF2: "o",           // GREEK SMALL LETTER OMEGA WITH VARIA AND YPOGEGRAMMENI
	0x1FF3: "o",           // GREEK SMALL LETTER OMEGA WITH YPOGEGRAMMENI
	0x1FF4: "o",           // GREEK SMALL LETTER OMEGA WITH OXIA AND YPOGEGRAMMENI
	0x1FF6: "o",           // GREEK SMALL LETTER OMEGA WITH PERISPOMENI
	0x1FF7: "o",           // GREEK SMALL LETTER OMEGA WITH PERISPOMENI AND YPOGEGRAMMENI
	0x1FF8: "O",           // GREEK CAPITAL LETTER OMICRON WITH VARIA
	0x1FF9: "O",           // GREEK CAPITAL LETTER OMICRON WITH OXIA
	0x1FFA: "O",           // GREEK CAPITAL LETTER OMEGA WITH VARIA
	0x1FFB: "O",           // GREEK CAPITAL LETTER OMEGA WITH OXIA
	0x1FFC: "O",           // GREEK CAPITAL LETTER OMEGA WITH PROSGEGRAMMENI
	0x1FFD: "'",           // GREEK OXIA
	0x1FFE: "`",           // GREEK DASIA
	0x2010: "-",           // HYPHEN
	0x2011: "-",           // NON-BREAKING HYPHEN
	0x2012: "-",           // FIGURE DASH
	0x2013: "-",           // EN DASH
	0x2014: "--",          // EM DASH
	0x2015: "--",          // HORIZONTAL BAR
	0x2016: "||",          // DOUBLE VERTICAL LINE
	0x2017: "_",           // DOUBLE LOW LINE
	0x2018: "'",           // LEFT SINGLE QUOTATION MARK
	0x2019: "'",           // RIGHT SINGLE QUOTATION MARK
	0x201A: ",",           // SINGLE LOW-9 QUOTATION MARK
	0x201B: "'",           // SINGLE HIGH-REVERSED-9 QUOTATION MARK
	0x201C: "\"",          // LEFT DOUBLE QUOTATION MARK
	0x201D: "\"",          // RIGHT DOUBLE QUOTATION MARK
	0x201E: "\"",          // DOUBLE LOW-9 QUOTATION MARK
	0x201F: "\"",          // DOUBLE HIGH-REVERSED-9 QUOTATION MARK
	0x2020: "+",           // DAGGER
	0x2021: "++",          // DOUBLE DAGGER
	0x2022: "*",           // BULLET
	0x2023: "*>",          // TRIANGULAR BULLET
	0x2024: ".",           // ONE DOT LEADER
	0x2025: "..",          // TWO DOT LEADER
	0x2026: "...",         // HORIZONTAL ELLIPSIS
	0x2027: ".",           // HYPHENATION POINT
	0x2030: "%0",          // PER MILLE SIGN
	0x2031: "%00",         // PER TEN THOUSAND SIGN
	0x2032: "'",           // PRIME
	0x2033: "''",          // DOUBLE PRIME
	0x2034: "'''",         // TRIPLE PRIME
	0x2035: "`",           // REVERSED PRIME
	0x2036: "``",          // REVERSED DOUBLE PRIME
	0x2037: "```",         // REVERSED TRIPLE PRIME
	0x2038: "^",           // CARET
	0x203B: "*",           // REFERENCE MARK
	0x203C: "!!",          // DOUBLE EXCLAMATION MARK
	0x203D: "!?",          // INTERROBANG
	0x203E: "-",           // OVERLINE
	0x203F: "_",           // UNDERTIE
	0x2040: "-",           // CHARACTER TIE
	0x2041: "^",           // CARET INSERTION POINT
	0x2042: "***",         // ASTERISM
	0x2043: "--",          // HYPHEN BULLET
	0x2045: "-[",          // LEFT SQUARE BRACKET WITH QUILL
	0x2046: "]-",          // RIGHT SQUARE BRACKET WITH QUILL
	0x2047: "??",          //
	0x2048: "?!",          // QUESTION EXCLAMATION MARK
	0x2049: "!?",          // EXCLAMATION QUESTION MARK
	0x204A: "7",           // TIRONIAN SIGN ET
	0x204B: "PP",          // REVERSED PILCROW SIGN
	0x204C: "(]",          // BLACK LEFTWARDS BULLET
	0x204D: "[)",          // BLACK RIGHTWARDS BULLET
	0x204E: "*",           //
	0x2052: "%",           //
	0x2053: "~",           //
	0x206F: "0",           // NOMINAL DIGIT SHAPES
	0x2070: "(0)",         // SUPERSCRIPT ZERO
	0x2071: "(i)",         //
	0x2074: "(4)",         // SUPERSCRIPT FOUR
	0x2075: "(5)",         // SUPERSCRIPT FIVE
	0x2076: "(6)",         // SUPERSCRIPT SIX
	0x2077: "(7)",         // SUPERSCRIPT SEVEN
	0x2078: "(8)",         // SUPERSCRIPT EIGHT
	0x2079: "(9)",         // SUPERSCRIPT NINE
	0x207A: "(+)",         // SUPERSCRIPT PLUS SIGN
	0x207B: "(-)",         // SUPERSCRIPT MINUS
	0x207C: "(=)",         // SUPERSCRIPT EQUALS SIGN
	0x207D: "(()",         // SUPERSCRIPT LEFT PARENTHESIS
	0x207E: "())",         // SUPERSCRIPT RIGHT PARENTHESIS
	0x207F: "(n)",         // SUPERSCRIPT LATIN SMALL LETTER N
	0x2080: "(0)",         // SUBSCRIPT ZERO
	0x2081: "(1)",         // SUBSCRIPT ONE
	0x2082: "(2)",         // SUBSCRIPT TWO
	0x2083: "(3)",         // SUBSCRIPT THREE
	0x2084: "(4)",         // SUBSCRIPT FOUR
	0x2085: "(5)",         // SUBSCRIPT FIVE
	0x2086: "(6)",         // SUBSCRIPT SIX
	0x2087: "(7)",         // SUBSCRIPT SEVEN
	0x2088: "(8)",         // SUBSCRIPT EIGHT
	0x2089: "(9)",         // SUBSCRIPT NINE
	0x208A: "(+)",         // SUBSCRIPT PLUS SIGN
	0x208B: "(-)",         // SUBSCRIPT MINUS
	0x208C: "(=)",         // SUBSCRIPT EQUALS SIGN
	0x208D: "(()",         // SUBSCRIPT LEFT PARENTHESIS
	0x208E: "())",         // SUBSCRIPT RIGHT PARENTHESIS
	0x2090: "(a)",         //
	0x2091: "(e)",         //
	0x2092: "(o)",         //
	0x2093: "(x)",         //
	0x2095: "(h)",         //
	0x2096: "(k)",         //
	0x2097: "(l)",         //
	0x2098: "(m)",         //
	0x2099: "(n)",         //
	0x209A: "(p)",         //
	0x209B: "(s)",         //
	0x209C: "(t)",         //
	0x3041: "a",           // HIRAGANA LETTER SMALL A
	0x3042: "a",           // HIRAGANA LETTER A
	0x3043: "i",           // HIRAGANA LETTER SMALL I
	0x3044: "i",           // HIRAGANA LETTER I
	0x3045: "u",           // HIRAGANA LETTER SMALL U
	0x3046: "u",           // HIRAGANA LETTER U
	0x3047: "e",           // HIRAGANA LETTER SMALL E
	0x3048: "e",           // HIRAGANA LETTER E
	0x3049: "o",           // HIRAGANA LETTER SMALL O
	0x304A: "o",           // HIRAGANA LETTER O
	0x304B: "ka",          // HIRAGANA LETTER KA
	0x304C: "ga",          // HIRAGANA LETTER GA
	0x304D: "ki",          // HIRAGANA LETTER KI
	0x304E: "gi",          // HIRAGANA LETTER GI
	0x304F: "ku",          // HIRAGANA LETTER KU
	0x3050: "gu",          // HIRAGANA LETTER GU
	0x3051: "ke",          // HIRAGANA LETTER KE
	0x3052: "ge",          // HIRAGANA LETTER GE
	0x3053: "ko",          // HIRAGANA LETTER KO
	0x3054: "go",          // HIRAGANA LETTER GO
	0x3055: "sa",          // HIRAGANA LETTER SA
	0x3056: "za",          // HIRAGANA LETTER ZA
	0x3057: "shi",         // HIRAGANA LETTER SI
	0x3058: "zi",          // HIRAGANA LETTER ZI
	0x3059: "su",          // HIRAGANA LETTER SU
	0x305A: "zu",          // HIRAGANA LETTER ZU
	0x305B: "se",          // HIRAGANA LETTER SE
	0x305C: "ze",          // HIRAGANA LETTER ZE
	0x305D: "so",          // HIRAGANA LETTER SO
	0x305E: "zo",          // HIRAGANA LETTER ZO
	0x305F: "ta",          // HIRAGANA LETTER TA
	0x3060: "da",          // HIRAGANA LETTER DA
	0x3061: "chi",         // HIRAGANA LETTER TI
	0x3062: "di",          // HIRAGANA LETTER DI
	0x3063: "tsu",         // HIRAGANA LETTER SMALL TU
	0x3064: "tsu",         // HIRAGANA LETTER TU
	0x3065: "du",          // HIRAGANA LETTER DU
	0x3066: "te",          // HIRAGANA LETTER TE
	0x3067: "de",          // HIRAGANA LETTER DE
	0x3068: "to",          // HIRAGANA LETTER TO
	0x3069: "do",          // HIRAGANA LETTER DO
	0x306A: "na",          // HIRAGANA LETTER NA
	0x306B: "ni",          // HIRAGANA LETTER NI
	0x306C: "nu",          // HIRAGANA LETTER NU
	0x306D: "ne",          // HIRAGANA LETTER NE
	0x306E: "no",          // HIRAGANA LETTER NO
	0x306F: "ha",          // HIRAGANA LETTER HA
	0x3070: "ba",          // HIRAGANA LETTER BA
	0x3071: "pa",          // HIRAGANA LETTER PA
	0x3072: "hi",          // HIRAGANA LETTER HI
	0x3073: "bi",          // HIRAGANA LETTER BI
	0x3074: "pi",          // HIRAGANA LETTER PI
	0x3075: "hu",          // HIRAGANA LETTER HU
	0x3076: "bu",          // HIRAGANA LETTER BU
	0x3077: "pu",          // HIRAGANA LETTER PU
	0x3078: "he",          // HIRAGANA LETTER HE
	0x3079: "be",          // HIRAGANA LETTER BE
	0x307A: "pe",          // HIRAGANA LETTER PE
	0x307B: "ho",          // HIRAGANA LETTER HO
	0x307C: "bo",          // HIRAGANA LETTER BO
	0x307D: "po",          // HIRAGANA LETTER PO
	0x307E: "ma",          // HIRAGANA LETTER MA
	0x307F: "mi",          // HIRAGANA LETTER MI
	0x3080: "mu",          // HIRAGANA LETTER MU
	0x3081: "me",          // HIRAGANA LETTER ME
	0x3082: "mo",          // HIRAGANA LETTER MO
	0x3083: "ya",          // HIRAGANA LETTER SMALL YA
	0x3084: "ya",          // HIRAGANA LETTER YA
	0x3085: "yu",          // HIRAGANA LETTER SMALL YU
	0x3086: "yu",          // HIRAGANA LETTER YU
	0x3087: "yo",          // HIRAGANA LETTER SMALL YO
	0x3088: "yo",          // HIRAGANA LETTER YO
	0x3089: "ra",          // HIRAGANA LETTER RA
	0x308A: "ri",          // HIRAGANA LETTER RI
	0x308B: "ru",          // HIRAGANA LETTER RU
	0x308C: "re",          // HIRAGANA LETTER RE
	0x308D: "ro",          // HIRAGANA LETTER RO
	0x308E: "wa",          // HIRAGANA LETTER SMALL WA
	0x308F: "wa",          // HIRAGANA LETTER WA
	0x3090: "wi",          // HIRAGANA LETTER WI
	0x3091: "we",          // HIRAGANA LETTER WE
	0x3092: "wo",          // HIRAGANA LETTER WO
	0x3093: "n",           // HIRAGANA LETTER N
	0x3094: "vu",          // HIRAGANA LETTER VU
	0x309D: "\"",          // HIRAGANA ITERATION MARK
	0x309E: "\"",          // HIRAGANA VOICED ITERATION MARK
	0x30A1: "a",           // KATAKANA LETTER SMALL A
	0x30A2: "a",           // KATAKANA LETTER A
	0x30A3: "i",           // KATAKANA LETTER SMALL I
	0x30A4: "i",           // KATAKANA LETTER I
	0x30A5: "u",           // KATAKANA LETTER SMALL U
	0x30A6: "u",           // KATAKANA LETTER U
	0x30A7: "e",           // KATAKANA LETTER SMALL E
	0x30A8: "e",           // KATAKANA LETTER E
	0x30A9: "o",           // KATAKANA LETTER SMALL O
	0x30AA: "o",           // KATAKANA LETTER O
	0x30AB: "ka",          // KATAKANA LETTER KA
	0x30AC: "ga",          // KATAKANA LETTER GA
	0x30AD: "ki",          // KATAKANA LETTER KI
	0x30AE: "gi",          // KATAKANA LETTER GI
	0x30AF: "ku",          // KATAKANA LETTER KU
	0x30B0: "gu",          // KATAKANA LETTER GU
	0x30B1: "ke",          // KATAKANA LETTER KE
	0x30B2: "ge",          // KATAKANA LETTER GE
	0x30B3: "ko",          // KATAKANA LETTER KO
	0x30B4: "go",          // KATAKANA LETTER GO
	0x30B5: "sa",          // KATAKANA LETTER SA
	0x30B6: "za",          // KATAKANA LETTER ZA
	0x30B7: "shi",         // KATAKANA LETTER SI
	0x30B8: "zi",          // KATAKANA LETTER ZI
	0x30B9: "su",          // KATAKANA LETTER SU
	0x30BA: "zu",          // KATAKANA LETTER ZU
	0x30BB: "se",          // KATAKANA LETTER SE
	0x30BC: "ze",          // KATAKANA LETTER ZE
	0x30BD: "so",          // KATAKANA LETTER SO
	0x30BE: "zo",          // KATAKANA LETTER ZO
	0x30BF: "ta",          // KATAKANA LETTER TA
	0x30C0: "da",          // KATAKANA LETTER DA
	0x30C1: "chi",         // KATAKANA LETTER TI
	0x30C2: "di",          // KATAKANA LETTER DI
	0x30C3: "tsu",         // KATAKANA LETTER SMALL TU
	0x30C4: "tsu",         // KATAKANA LETTER TU
	0x30C5: "du",          // KATAKANA LETTER DU
	0x30C6: "te",          // KATAKANA LETTER TE
	0x30C7: "de",          // KATAKANA LETTER DE
	0x30C8: "to",          // KATAKANA LETTER TO
	0x30C9: "do",          // KATAKANA LETTER DO
	0x30CA: "na",          // KATAKANA LETTER NA
	0x30CB: "ni",          // KATAKANA LETTER NI
	0x30CC: "nu",          // KATAKANA LETTER NU
	0x30CD: "ne",          // KATAKANA LETTER NE
	0x30CE: "no",          // KATAKANA LETTER NO
	0x30CF: "ha",          // KATAKANA LETTER HA
	0x30D0: "ba",          // KATAKANA LETTER BA
	0x30D1: "pa",          // KATAKANA LETTER PA
	0x30D2: "hi",          // KATAKANA LETTER HI
	0x30D3: "bi",          // KATAKANA LETTER BI
	0x30D4: "pi",          // KATAKANA LETTER PI
	0x30D5: "hu",          // KATAKANA LETTER HU
	0x30D6: "bu",          // KATAKANA LETTER BU
	0x30D7: "pu",          // KATAKANA LETTER PU
	0x30D8: "he",          // KATAKANA LETTER HE
	0x30D9: "be",          // KATAKANA LETTER BE
	0x30DA: "pe",          // KATAKANA LETTER PE
	0x30DB: "ho",          // KATAKANA LETTER HO
	0x30DC: "bo",          // KATAKANA LETTER BO
	0x30DD: "po",          // KATAKANA LETTER PO
	0x30DE: "ma",          // KATAKANA LETTER MA
	0x30DF: "mi",          // KATAKANA LETTER MI
	0x30E0: "mu",          // KATAKANA LETTER MU
	0x30E1: "me",          // KATAKANA LETTER ME
	0x30E2: "mo",          // KATAKANA LETTER MO
	0x30E3: "ya",          // KATAKANA LETTER SMALL YA
	0x30E4: "ya",          // KATAKANA LETTER YA
	0x30E5: "yu",          // KATAKANA LETTER SMALL YU
	0x30E6: "yu",          // KATAKANA LETTER YU
	0x30E7: "yo",          // KATAKANA LETTER SMALL YO
	0x30E8: "yo",          // KATAKANA LETTER YO
	0x30E9: "ra",          // KATAKANA LETTER RA
	0x30EA: "ri",          // KATAKANA LETTER RI
	0x30EB: "ru",          // KATAKANA LETTER RU
	0x30EC: "re",          // KATAKANA LETTER RE
	0x30ED: "ro",          // KATAKANA LETTER RO
	0x30EE: "wa",          // KATAKANA LETTER SMALL WA
	0x30EF: "wa",          // KATAKANA LETTER WA
	0x30F0: "wi",          // KATAKANA LETTER WI
	0x30F1: "we",          // KATAKANA LETTER WE
	0x30F2: "wo",          // KATAKANA LETTER WO
	0x30F3: "n",           // KATAKANA LETTER N
	0x30F4: "vu",          // KATAKANA LETTER VU
	0x30F5: "ka",          // KATAKANA LETTER SMALL KA
	0x30F6: "ke",          // KATAKANA LETTER SMALL KE
	0x30F7: "va",          // KATAKANA LETTER VA
	0x30F8: "vi",          // KATAKANA LETTER VI
	0x30F9: "ve",          // KATAKANA LETTER VE
	0x30FA: "vo",          // KATAKANA LETTER VO
	0x30FD: "\\",          // KATAKANA ITERATION MARK
	0x30FE: "\\",          // KATAKANA VOICED ITERATION MARK
	0x3105: "B",           // BOPOMOFO LETTER B
	0x3106: "P",           // BOPOMOFO LETTER P
	0x3107: "M",           // BOPOMOFO LETTER M
	0x3108: "F",           // BOPOMOFO LETTER F
	0x3109: "D",           // BOPOMOFO LETTER D
	0x310A: "T",           // BOPOMOFO LETTER T
	0x310B: "N",           // BOPOMOFO LETTER N
	0x310C: "L",           // BOPOMOFO LETTER L
	0x310D: "G",           // BOPOMOFO LETTER G
	0x310E: "K",           // BOPOMOFO LETTER K
	0x310F: "H",           // BOPOMOFO LETTER H
	0x3110: "J",           // BOPOMOFO LETTER J
	0x3111: "Q",           // BOPOMOFO LETTER Q
	0x3112: "X",           // BOPOMOFO LETTER X
	0x3113: "ZH",          // BOPOMOFO LETTER ZH
	0x3114: "CH",          // BOPOMOFO LETTER CH
	0x3115: "SH",          // BOPOMOFO LETTER SH
	0x3116: "R",           // BOPOMOFO LETTER R
	0x3117: "Z",           // BOPOMOFO LETTER Z
	0x3118: "C",           // BOPOMOFO LETTER C
	0x3119: "S",           // BOPOMOFO LETTER S
	0x311A: "A",           // BOPOMOFO LETTER A
	0x311B: "O",           // BOPOMOFO LETTER O
	0x311C: "E",           // BOPOMOFO LETTER E
	0x311D: "EH",          // BOPOMOFO LETTER EH
	0x311E: "AI",          // BOPOMOFO LETTER AI
	0x311F: "EI",          // BOPOMOFO LETTER EI
	0x3120: "AU",          // BOPOMOFO LETTER AU
	0x3121: "OU",          // BOPOMOFO LETTER OU
	0x3122: "AN",          // BOPOMOFO LETTER AN
	0x3123: "EN",          // BOPOMOFO LETTER EN
	0x3124: "ANG",         // BOPOMOFO LETTER ANG
	0x3125: "ENG",         // BOPOMOFO LETTER ENG
	0x3126: "ER",          // BOPOMOFO LETTER ER
	0x3127: "I",           // BOPOMOFO LETTER I
	0x3128: "U",           // BOPOMOFO LETTER U
	0x3129: "IU",          // BOPOMOFO LETTER IU
	0x312A: "V",           // BOPOMOFO LETTER V
	0x312B: "NG",          // BOPOMOFO LETTER NG
	0x312C: "GN",          // BOPOMOFO LETTER GN
	0x3131: "g",           // HANGUL LETTER KIYEOK
	0x3132: "gg",          // HANGUL LETTER SSANGKIYEOK
	0x3133: "gs",          // HANGUL LETTER KIYEOK-SIOS
	0x3134: "n",           // HANGUL LETTER NIEUN
	0x3135: "nj",          // HANGUL LETTER NIEUN-CIEUC
	0x3136: "nh",          // HANGUL LETTER NIEUN-HIEUH
	0x3137: "d",           // HANGUL LETTER TIKEUT
	0x3138: "dd",          // HANGUL LETTER SSANGTIKEUT
	0x3139: "r",           // HANGUL LETTER RIEUL
	0x313A: "lg",          // HANGUL LETTER RIEUL-KIYEOK
	0x313B: "lm",          // HANGUL LETTER RIEUL-MIEUM
	0x313C: "lb",          // HANGUL LETTER RIEUL-PIEUP
	0x313D: "ls",          // HANGUL LETTER RIEUL-SIOS
	0x313E: "lt",          // HANGUL LETTER RIEUL-THIEUTH
	0x313F: "lp",          // HANGUL LETTER RIEUL-PHIEUPH
	0x3140: "rh",          // HANGUL LETTER RIEUL-HIEUH
	0x3141: "m",           // HANGUL LETTER MIEUM
	0x3142: "b",           // HANGUL LETTER PIEUP
	0x3143: "bb",          // HANGUL LETTER SSANGPIEUP
	0x3144: "bs",          // HANGUL LETTER PIEUP-SIOS
	0x3145: "s",           // HANGUL LETTER SIOS
	0x3146: "ss",          // HANGUL LETTER SSANGSIOS
	0x3147: "o",           // HANGUL LETTER IEUNG
	0x3148: "j",           // HANGUL LETTER CIEUC
	0x3149: "jj",          // HANGUL LETTER SSANGCIEUC
	0x314A: "c",           // HANGUL LETTER CHIEUCH
	0x314B: "k",           // HANGUL LETTER KHIEUKH
	0x314C: "t",           // HANGUL LETTER THIEUTH
	0x314D: "p",           // HANGUL LETTER PHIEUPH
	0x314E: "h",           // HANGUL LETTER HIEUH
	0x314F: "a",           // HANGUL LETTER A
	0x3150: "ae",          // HANGUL LETTER AE
	0x3151: "ya",          // HANGUL LETTER YA
	0x3152: "yae",         // HANGUL LETTER YAE
	0x3153: "eo",          // HANGUL LETTER EO
	0x3154: "e",           // HANGUL LETTER E
	0x3155: "yeo",         // HANGUL LETTER YEO
	0x3156: "ye",          // HANGUL LETTER YE
	0x3157: "o",           // HANGUL LETTER O
	0x3158: "wa",          // HANGUL LETTER WA
	0x3159: "wae",         // HANGUL LETTER WAE
	0x315A: "oe",          // HANGUL LETTER OE
	0x315B: "yo",          // HANGUL LETTER YO
	0x315C: "u",           // HANGUL LETTER U
	0x315D: "weo",         // HANGUL LETTER WEO
	0x315E: "we",          // HANGUL LETTER WE
	0x315F: "wi",          // HANGUL LETTER WI
	0x3160: "yu",          // HANGUL LETTER YU
	0x3161: "eu",          // HANGUL LETTER EU
	0x3162: "yi",          // HANGUL LETTER YI
	0x3163: "i",           // HANGUL LETTER I
	0x3165: "nn",          // HANGUL LETTER SSANGNIEUN
	0x3166: "nd",          // HANGUL LETTER NIEUN-TIKEUT
	0x3167: "ns",          // HANGUL LETTER NIEUN-SIOS
	0x3168: "nZ",          // HANGUL LETTER NIEUN-PANSIOS
	0x3169: "lgs",         // HANGUL LETTER RIEUL-KIYEOK-SIOS
	0x316A: "ld",          // HANGUL LETTER RIEUL-TIKEUT
	0x316B: "lbs",         // HANGUL LETTER RIEUL-PIEUP-SIOS
	0x316C: "lZ",          // HANGUL LETTER RIEUL-PANSIOS
	0x316D: "lQ",          // HANGUL LETTER RIEUL-YEORINHIEUH
	0x316E: "mb",          // HANGUL LETTER MIEUM-PIEUP
	0x316F: "ms",          // HANGUL LETTER MIEUM-SIOS
	0x3170: "mZ",          // HANGUL LETTER MIEUM-PANSIOS
	0x3171: "mN",          // HANGUL LETTER KAPYEOUNMIEUM
	0x3172: "bg",          // HANGUL LETTER PIEUP-KIYEOK
	0x3174: "bsg",         // HANGUL LETTER PIEUP-SIOS-KIYEOK
	0x3175: "bst",         // HANGUL LETTER PIEUP-SIOS-TIKEUT
	0x3176: "bj",          // HANGUL LETTER PIEUP-CIEUC
	0x3177: "bt",          // HANGUL LETTER PIEUP-THIEUTH
	0x3178: "bN",          // HANGUL LETTER KAPYEOUNPIEUP
	0x3179: "bbN",         // HANGUL LETTER KAPYEOUNSSANGPIEUP
	0x317A: "sg",          // HANGUL LETTER SIOS-KIYEOK
	0x317B: "sn",          // HANGUL LETTER SIOS-NIEUN
	0x317C: "sd",          // HANGUL LETTER SIOS-TIKEUT
	0x317D: "sb",          // HANGUL LETTER SIOS-PIEUP
	0x317E: "sj",          // HANGUL LETTER SIOS-CIEUC
	0x317F: "Z",           // HANGUL LETTER PANSIOS
	0x3181: "N",           // HANGUL LETTER YESIEUNG
	0x3182: "Ns",          // HANGUL LETTER YESIEUNG-SIOS
	0x3183: "NZ",          // HANGUL LETTER YESIEUNG-PANSIOS
	0x3184: "pN",          // HANGUL LETTER KAPYEOUNPHIEUPH
	0x3185: "hh",          // HANGUL LETTER SSANGHIEUH
	0x3186: "Q",           // HANGUL LETTER YEORINHIEUH
	0x3187: "yo-ya",       // HANGUL LETTER YO-YA
	0x3188: "yo-yae",      // HANGUL LETTER YO-YAE
	0x3189: "yo-i",        // HANGUL LETTER YO-I
	0x318A: "yu-yeo",      // HANGUL LETTER YU-YEO
	0x318B: "yu-ye",       // HANGUL LETTER YU-YE
	0x318C: "yu-i",        // HANGUL LETTER YU-I
	0x318D: "U",           // HANGUL LETTER ARAEA
	0x318E: "U-i",         // HANGUL LETTER ARAEAE
	0x31A0: "BU",          // BOPOMOFO LETTER BU
	0x31A1: "ZI",          // BOPOMOFO LETTER ZI
	0x31A2: "JI",          // BOPOMOFO LETTER JI
	0x31A3: "GU",          // BOPOMOFO LETTER GU
	0x31A4: "EE",          // BOPOMOFO LETTER EE
	0x31A5: "ENN",         // BOPOMOFO LETTER ENN
	0x31A6: "OO",          // BOPOMOFO LETTER OO
	0x31A7: "ONN",         // BOPOMOFO LETTER ONN
	0x31A8: "IR",          // BOPOMOFO LETTER IR
	0x31A9: "ANN",         // BOPOMOFO LETTER ANN
	0x31AA: "INN",         // BOPOMOFO LETTER INN
	0x31AB: "UNN",         // BOPOMOFO LETTER UNN
	0x31AC: "IM",          // BOPOMOFO LETTER IM
	0x31AD: "NGG",         // BOPOMOFO LETTER NGG
	0x31AE: "AINN",        // BOPOMOFO LETTER AINN
	0x31AF: "AUNN",        // BOPOMOFO LETTER AUNN
	0x31B0: "AM",          // BOPOMOFO LETTER AM
	0x31B1: "OM",          // BOPOMOFO LETTER OM
	0x31B2: "ONG",         // BOPOMOFO LETTER ONG
	0x31B3: "INNN",        // BOPOMOFO LETTER INNN
	0x31B4: "P",           // BOPOMOFO FINAL LETTER P
	0x31B5: "T",           // BOPOMOFO FINAL LETTER T
	0x31B6: "K",           // BOPOMOFO FINAL LETTER K
	0x31B7: "H",           // BOPOMOFO FINAL LETTER H
	0x3200: "(g)",         // PARENTHESIZED HANGUL KIYEOK
	0x3201: "(n)",         // PARENTHESIZED HANGUL NIEUN
	0x3202: "(d)",         // PARENTHESIZED HANGUL TIKEUT
	0x3203: "(r)",         // PARENTHESIZED HANGUL RIEUL
	0x3204: "(m)",         // PARENTHESIZED HANGUL MIEUM
	0x3205: "(b)",         // PARENTHESIZED HANGUL PIEUP
	0x3206: "(s)",         // PARENTHESIZED HANGUL SIOS
	0x3207: "()",          // PARENTHESIZED HANGUL IEUNG
	0x3208: "(j)",         // PARENTHESIZED HANGUL CIEUC
	0x3209: "(c)",         // PARENTHESIZED HANGUL CHIEUCH
	0x320A: "(k)",         // PARENTHESIZED HANGUL KHIEUKH
	0x320B: "(t)",         // PARENTHESIZED HANGUL THIEUTH
	0x320C: "(p)",         // PARENTHESIZED HANGUL PHIEUPH
	0x320D: "(h)",         // PARENTHESIZED HANGUL HIEUH
	0x320E: "(ga)",        // PARENTHESIZED HANGUL KIYEOK A
	0x320F: "(na)",        // PARENTHESIZED HANGUL NIEUN A
	0x3210: "(da)",        // PARENTHESIZED HANGUL TIKEUT A
	0x3211: "(ra)",        // PARENTHESIZED HANGUL RIEUL A
	0x3212: "(ma)",        // PARENTHESIZED HANGUL MIEUM A
	0x3213: "(ba)",        // PARENTHESIZED HANGUL PIEUP A
	0x3214: "(sa)",        // PARENTHESIZED HANGUL SIOS A
	0x3215: "(a)",         // PARENTHESIZED HANGUL IEUNG A
	0x3216: "(ja)",        // PARENTHESIZED HANGUL CIEUC A
	0x3217: "(ca)",        // PARENTHESIZED HANGUL CHIEUCH A
	0x3218: "(ka)",        // PARENTHESIZED HANGUL KHIEUKH A
	0x3219: "(ta)",        // PARENTHESIZED HANGUL THIEUTH A
	0x321A: "(pa)",        // PARENTHESIZED HANGUL PHIEUPH A
	0x321B: "(ha)",        // PARENTHESIZED HANGUL HIEUH A
	0x321C: "(ju)",        // PARENTHESIZED HANGUL CIEUC U
	0x3220: "(1) ",        // PARENTHESIZED IDEOGRAPH ONE
	0x3221: "(2) ",        // PARENTHESIZED IDEOGRAPH TWO
	0x3222: "(3) ",        // PARENTHESIZED IDEOGRAPH THREE
	0x3223: "(4) ",        // PARENTHESIZED IDEOGRAPH FOUR
	0x3224: "(5) ",        // PARENTHESIZED IDEOGRAPH FIVE
	0x3225: "(6) ",        // PARENTHESIZED IDEOGRAPH SIX
	0x3226: "(7) ",        // PARENTHESIZED IDEOGRAPH SEVEN
	0x3227: "(8) ",        // PARENTHESIZED IDEOGRAPH EIGHT
	0x3228: "(9) ",        // PARENTHESIZED IDEOGRAPH NINE
	0x3229: "(10) ",       // PARENTHESIZED IDEOGRAPH TEN
	0x322A: "(Yue) ",      // PARENTHESIZED IDEOGRAPH MOON
	0x322B: "(Huo) ",      // PARENTHESIZED IDEOGRAPH FIRE
	0x322C: "(Shui) ",     // PARENTHESIZED IDEOGRAPH WATER
	0x322D: "(Mu) ",       // PARENTHESIZED IDEOGRAPH WOOD
	0x322E: "(Jin) ",      // PARENTHESIZED IDEOGRAPH METAL
	0x322F: "(Tu) ",       // PARENTHESIZED IDEOGRAPH EARTH
	0x3230: "(Ri) ",       // PARENTHESIZED IDEOGRAPH SUN
	0x3231: "(Zhu) ",      // PARENTHESIZED IDEOGRAPH STOCK
	0x3232: "(You) ",      // PARENTHESIZED IDEOGRAPH HAVE
	0x3233: "(She) ",      // PARENTHESIZED IDEOGRAPH SOCIETY
	0x3234: "(Ming) ",     // PARENTHESIZED IDEOGRAPH NAME
	0x3235: "(Te) ",       // PARENTHESIZED IDEOGRAPH SPECIAL
	0x3236: "(Cai) ",      // PARENTHESIZED IDEOGRAPH FINANCIAL
	0x3237: "(Zhu) ",      // PARENTHESIZED IDEOGRAPH CONGRATULATION
	0x3238: "(Lao) ",      // PARENTHESIZED IDEOGRAPH LABOR
	0x3239: "(Dai) ",      // PARENTHESIZED IDEOGRAPH REPRESENT
	0x323A: "(Hu) ",       // PARENTHESIZED IDEOGRAPH CALL
	0x323B: "(Xue) ",      // PARENTHESIZED IDEOGRAPH STUDY
	0x323C: "(Jian) ",     // PARENTHESIZED IDEOGRAPH SUPERVISE
	0x323D: "(Qi) ",       // PARENTHESIZED IDEOGRAPH ENTERPRISE
	0x323E: "(Zi) ",       // PARENTHESIZED IDEOGRAPH RESOURCE
	0x323F: "(Xie) ",      // PARENTHESIZED IDEOGRAPH ALLIANCE
	0x3240: "(Ji) ",       // PARENTHESIZED IDEOGRAPH FESTIVAL
	0x3241: "(Xiu) ",      // PARENTHESIZED IDEOGRAPH REST
	0x3242: "<<",          // PARENTHESIZED IDEOGRAPH SELF
	0x3243: ">>",          // PARENTHESIZED IDEOGRAPH REACH
	0x3251: "21",          //
	0x3252: "22",          //
	0x3253: "23",          //
	0x3254: "24",          //
	0x3255: "25",          //
	0x3256: "26",          //
	0x3257: "27",          //
	0x3258: "28",          //
	0x3259: "29",          //
	0x325A: "30",          //
	0x325B: "31",          //
	0x325C: "32",          //
	0x325D: "33",          //
	0x325E: "34",          //
	0x325F: "35",          //
	0x3260: "(g)",         // CIRCLED HANGUL KIYEOK
	0x3261: "(n)",         // CIRCLED HANGUL NIEUN
	0x3262: "(d)",         // CIRCLED HANGUL TIKEUT
	0x3263: "(r)",         // CIRCLED HANGUL RIEUL
	0x3264: "(m)",         // CIRCLED HANGUL MIEUM
	0x3265: "(b)",         // CIRCLED HANGUL PIEUP
	0x3266: "(s)",         // CIRCLED HANGUL SIOS
	0x3267: "()",          // CIRCLED HANGUL IEUNG
	0x3268: "(j)",         // CIRCLED HANGUL CIEUC
	0x3269: "(c)",         // CIRCLED HANGUL CHIEUCH
	0x326A: "(k)",         // CIRCLED HANGUL KHIEUKH
	0x326B: "(t)",         // CIRCLED HANGUL THIEUTH
	0x326C: "(p)",         // CIRCLED HANGUL PHIEUPH
	0x326D: "(h)",         // CIRCLED HANGUL HIEUH
	0x326E: "(ga)",        // CIRCLED HANGUL KIYEOK A
	0x326F: "(na)",        // CIRCLED HANGUL NIEUN A
	0x3270: "(da)",        // CIRCLED HANGUL TIKEUT A
	0x3271: "(ra)",        // CIRCLED HANGUL RIEUL A
	0x3272: "(ma)",        // CIRCLED HANGUL MIEUM A
	0x3273: "(ba)",        // CIRCLED HANGUL PIEUP A
	0x3274: "(sa)",        // CIRCLED HANGUL SIOS A
	0x3275: "(a)",         // CIRCLED HANGUL IEUNG A
	0x3276: "(ja)",        // CIRCLED HANGUL CIEUC A
	0x3277: "(ca)",        // CIRCLED HANGUL CHIEUCH A
	0x3278: "(ka)",        // CIRCLED HANGUL KHIEUKH A
	0x3279: "(ta)",        // CIRCLED HANGUL THIEUTH A
	0x327A: "(pa)",        // CIRCLED HANGUL PHIEUPH A
	0x327B: "(ha)",        // CIRCLED HANGUL HIEUH A
	0x327F: "KIS ",        // KOREAN STANDARD SYMBOL
	0x3280: "(1) ",        // CIRCLED IDEOGRAPH ONE
	0x3281: "(2) ",        // CIRCLED IDEOGRAPH TWO
	0x3282: "(3) ",        // CIRCLED IDEOGRAPH THREE
	0x3283: "(4) ",        // CIRCLED IDEOGRAPH FOUR
	0x3284: "(5) ",        // CIRCLED IDEOGRAPH FIVE
	0x3285: "(6) ",        // CIRCLED IDEOGRAPH SIX
	0x3286: "(7) ",        // CIRCLED IDEOGRAPH SEVEN
	0x3287: "(8) ",        // CIRCLED IDEOGRAPH EIGHT
	0x3288: "(9) ",        // CIRCLED IDEOGRAPH NINE
	0x3289: "(10) ",       // CIRCLED IDEOGRAPH TEN
	0x328A: "(Yue) ",      // CIRCLED IDEOGRAPH MOON
	0x328B: "(Huo) ",      // CIRCLED IDEOGRAPH FIRE
	0x328C: "(Shui) ",     // CIRCLED IDEOGRAPH WATER
	0x328D: "(Mu) ",       // CIRCLED IDEOGRAPH WOOD
	0x328E: "(Jin) ",      // CIRCLED IDEOGRAPH METAL
	0x328F: "(Tu) ",       // CIRCLED IDEOGRAPH EARTH
	0x3290: "(Ri) ",       // CIRCLED IDEOGRAPH SUN
	0x3291: "(Zhu) ",      // CIRCLED IDEOGRAPH STOCK
	0x3292: "(You) ",      // CIRCLED IDEOGRAPH HAVE
	0x3293: "(She) ",      // CIRCLED IDEOGRAPH SOCIETY
	0x3294: "(Ming) ",     // CIRCLED IDEOGRAPH NAME
	0x3295: "(Te) ",       // CIRCLED IDEOGRAPH SPECIAL
	0x3296: "(Cai) ",      // CIRCLED IDEOGRAPH FINANCIAL
	0x3297: "(Zhu) ",      // CIRCLED IDEOGRAPH CONGRATULATION
	0x3298: "(Lao) ",      // CIRCLED IDEOGRAPH LABOR
	0x3299: "(Mi) ",       // CIRCLED IDEOGRAPH SECRET
	0x329A: "(Nan) ",      // CIRCLED IDEOGRAPH MALE
	0x329B: "(Nu) ",       // CIRCLED IDEOGRAPH FEMALE
	0x329C: "(Shi) ",      // CIRCLED IDEOGRAPH SUITABLE
	0x329D: "(You) ",      // CIRCLED IDEOGRAPH EXCELLENT
	0x329E: "(Yin) ",      // CIRCLED IDEOGRAPH PRINT
	0x329F: "(Zhu) ",      // CIRCLED IDEOGRAPH ATTENTION
	0x32A0: "(Xiang) ",    // CIRCLED IDEOGRAPH ITEM
	0x32A1: "(Xiu) ",      // CIRCLED IDEOGRAPH REST
	0x32A2: "(Xie) ",      // CIRCLED IDEOGRAPH COPY
	0x32A3: "(Zheng) ",    // CIRCLED IDEOGRAPH CORRECT
	0x32A4: "(Shang) ",    // CIRCLED IDEOGRAPH HIGH
	0x32A5: "(Zhong) ",    // CIRCLED IDEOGRAPH CENTRE
	0x32A6: "(Xia) ",      // CIRCLED IDEOGRAPH LOW
	0x32A7: "(Zuo) ",      // CIRCLED IDEOGRAPH LEFT
	0x32A8: "(You) ",      // CIRCLED IDEOGRAPH RIGHT
	0x32A9: "(Yi) ",       // CIRCLED IDEOGRAPH MEDICINE
	0x32AA: "(Zong) ",     // CIRCLED IDEOGRAPH RELIGION
	0x32AB: "(Xue) ",      // CIRCLED IDEOGRAPH STUDY
	0x32AC: "(Jian) ",     // CIRCLED IDEOGRAPH SUPERVISE
	0x32AD: "(Qi) ",       // CIRCLED IDEOGRAPH ENTERPRISE
	0x32AE: "(Zi) ",       // CIRCLED IDEOGRAPH RESOURCE
	0x32AF: "(Xie) ",      // CIRCLED IDEOGRAPH ALLIANCE
	0x32B0: "(Ye) ",       // CIRCLED IDEOGRAPH NIGHT
	0x32B1: "36",          //
	0x32B2: "37",          //
	0x32B3: "38",          //
	0x32B4: "39",          //
	0x32B5: "40",          //
	0x32B6: "41",          //
	0x32B7: "42",          //
	0x32B8: "43",          //
	0x32B9: "44",          //
	0x32BA: "45",          //
	0x32BB: "46",          //
	0x32BC: "47",          //
	0x32BD: "48",          //
	0x32BE: "49",          //
	0x32BF: "50",          //
	0x32C0: "1M",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR JANUARY
	0x32C1: "2M",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR FEBRUARY
	0x32C2: "3M",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR MARCH
	0x32C3: "4M",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR APRIL
	0x32C4: "5M",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR MAY
	0x32C5: "6M",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR JUNE
	0x32C6: "7M",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR JULY
	0x32C7: "8M",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR AUGUST
	0x32C8: "9M",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR SEPTEMBER
	0x32C9: "10M",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR OCTOBER
	0x32CA: "11M",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR NOVEMBER
	0x32CB: "12M",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DECEMBER
	0x32D0: "a",           // CIRCLED KATAKANA A
	0x32D1: "i",           // CIRCLED KATAKANA I
	0x32D2: "u",           // CIRCLED KATAKANA U
	0x32D3: "u",           // CIRCLED KATAKANA E
	0x32D4: "o",           // CIRCLED KATAKANA O
	0x32D5: "ka",          // CIRCLED KATAKANA KA
	0x32D6: "ki",          // CIRCLED KATAKANA KI
	0x32D7: "ku",          // CIRCLED KATAKANA KU
	0x32D8: "ke",          // CIRCLED KATAKANA KE
	0x32D9: "ko",          // CIRCLED KATAKANA KO
	0x32DA: "sa",          // CIRCLED KATAKANA SA
	0x32DB: "si",          // CIRCLED KATAKANA SI
	0x32DC: "su",          // CIRCLED KATAKANA SU
	0x32DD: "se",          // CIRCLED KATAKANA SE
	0x32DE: "so",          // CIRCLED KATAKANA SO
	0x32DF: "ta",          // CIRCLED KATAKANA TA
	0x32E0: "ti",          // CIRCLED KATAKANA TI
	0x32E1: "tu",          // CIRCLED KATAKANA TU
	0x32E2: "te",          // CIRCLED KATAKANA TE
	0x32E3: "to",          // CIRCLED KATAKANA TO
	0x32E4: "na",          // CIRCLED KATAKANA NA
	0x32E5: "ni",          // CIRCLED KATAKANA NI
	0x32E6: "nu",          // CIRCLED KATAKANA NU
	0x32E7: "ne",          // CIRCLED KATAKANA NE
	0x32E8: "no",          // CIRCLED KATAKANA NO
	0x32E9: "ha",          // CIRCLED KATAKANA HA
	0x32EA: "hi",          // CIRCLED KATAKANA HI
	0x32EB: "hu",          // CIRCLED KATAKANA HU
	0x32EC: "he",          // CIRCLED KATAKANA HE
	0x32ED: "ho",          // CIRCLED KATAKANA HO
	0x32EE: "ma",          // CIRCLED KATAKANA MA
	0x32EF: "mi",          // CIRCLED KATAKANA MI
	0x32F0: "mu",          // CIRCLED KATAKANA MU
	0x32F1: "me",          // CIRCLED KATAKANA ME
	0x32F2: "mo",          // CIRCLED KATAKANA MO
	0x32F3: "ya",          // CIRCLED KATAKANA YA
	0x32F4: "yu",          // CIRCLED KATAKANA YU
	0x32F5: "yo",          // CIRCLED KATAKANA YO
	0x32F6: "ra",          // CIRCLED KATAKANA RA
	0x32F7: "ri",          // CIRCLED KATAKANA RI
	0x32F8: "ru",          // CIRCLED KATAKANA RU
	0x32F9: "re",          // CIRCLED KATAKANA RE
	0x32FA: "ro",          // CIRCLED KATAKANA RO
	0x32FB: "wa",          // CIRCLED KATAKANA WA
	0x32FC: "wi",          // CIRCLED KATAKANA WI
	0x32FD: "we",          // CIRCLED KATAKANA WE
	0x32FE: "wo",          // CIRCLED KATAKANA WO
	0x3300: "apartment",   // SQUARE APAATO
	0x3301: "alpha",       // SQUARE ARUHUA
	0x3302: "ampere",      // SQUARE ANPEA
	0x3303: "are",         // SQUARE AARU
	0x3304: "inning",      // SQUARE ININGU
	0x3305: "inch",        // SQUARE INTI
	0x3306: "won",         // SQUARE UON
	0x3307: "escudo",      // SQUARE ESUKUUDO
	0x3308: "acre",        // SQUARE EEKAA
	0x3309: "ounce",       // SQUARE ONSU
	0x330A: "ohm",         // SQUARE OOMU
	0x330B: "kai-ri",      // SQUARE KAIRI
	0x330C: "carat",       // SQUARE KARATTO
	0x330D: "calorie",     // SQUARE KARORII
	0x330E: "gallon",      // SQUARE GARON
	0x330F: "gamma",       // SQUARE GANMA
	0x3310: "giga",        // SQUARE GIGA
	0x3311: "guinea",      // SQUARE GINII
	0x3312: "curie",       // SQUARE KYURII
	0x3313: "guilder",     // SQUARE GIRUDAA
	0x3314: "kilo",        // SQUARE KIRO
	0x3315: "kilogram",    // SQUARE KIROGURAMU
	0x3316: "kilometer",   // SQUARE KIROMEETORU
	0x3317: "kilowatt",    // SQUARE KIROWATTO
	0x3318: "gram",        // SQUARE GURAMU
	0x3319: "gram ton",    // SQUARE GURAMUTON
	0x331A: "cruzeiro",    // SQUARE KURUZEIRO
	0x331B: "krone",       // SQUARE KUROONE
	0x331C: "case",        // SQUARE KEESU
	0x331D: "koruna",      // SQUARE KORUNA
	0x331E: "co-op",       // SQUARE KOOPO
	0x331F: "cycle",       // SQUARE SAIKURU
	0x3320: "centime",     // SQUARE SANTIIMU
	0x3321: "shilling",    // SQUARE SIRINGU
	0x3322: "centi",       // SQUARE SENTI
	0x3323: "cent",        // SQUARE SENTO
	0x3324: "dozen",       // SQUARE DAASU
	0x3325: "desi",        // SQUARE DESI
	0x3326: "dollar",      // SQUARE DORU
	0x3327: "ton",         // SQUARE TON
	0x3328: "nano",        // SQUARE NANO
	0x3329: "knot",        // SQUARE NOTTO
	0x332A: "heights",     // SQUARE HAITU
	0x332B: "percent",     // SQUARE PAASENTO
	0x332C: "parts",       // SQUARE PAATU
	0x332D: "barrel",      // SQUARE BAARERU
	0x332E: "piaster",     // SQUARE PIASUTORU
	0x332F: "picul",       // SQUARE PIKURU
	0x3330: "pico",        // SQUARE PIKO
	0x3331: "building",    // SQUARE BIRU
	0x3332: "farad",       // SQUARE HUARADDO
	0x3333: "feet",        // SQUARE HUIITO
	0x3334: "bushel",      // SQUARE BUSSYERU
	0x3335: "franc",       // SQUARE HURAN
	0x3336: "hectare",     // SQUARE HEKUTAARU
	0x3337: "peso",        // SQUARE PESO
	0x3338: "pfennig",     // SQUARE PENIHI
	0x3339: "hertz",       // SQUARE HERUTU
	0x333A: "pence",       // SQUARE PENSU
	0x333B: "page",        // SQUARE PEEZI
	0x333C: "beta",        // SQUARE BEETA
	0x333D: "point",       // SQUARE POINTO
	0x333E: "volt",        // SQUARE BORUTO
	0x333F: "hon",         // SQUARE HON
	0x3340: "pound",       // SQUARE PONDO
	0x3341: "hall",        // SQUARE HOORU
	0x3342: "horn",        // SQUARE HOON
	0x3343: "micro",       // SQUARE MAIKURO
	0x3344: "mile",        // SQUARE MAIRU
	0x3345: "mach",        // SQUARE MAHHA
	0x3346: "mark",        // SQUARE MARUKU
	0x3347: "mansion",     // SQUARE MANSYON
	0x3348: "micron",      // SQUARE MIKURON
	0x3349: "milli",       // SQUARE MIRI
	0x334A: "millibar",    // SQUARE MIRIBAARU
	0x334B: "mega",        // SQUARE MEGA
	0x334C: "megaton",     // SQUARE MEGATON
	0x334D: "meter",       // SQUARE MEETORU
	0x334E: "yard",        // SQUARE YAADO
	0x334F: "yard",        // SQUARE YAARU
	0x3350: "yuan",        // SQUARE YUAN
	0x3351: "liter",       // SQUARE RITTORU
	0x3352: "lira",        // SQUARE RIRA
	0x3353: "rupee",       // SQUARE RUPII
	0x3354: "ruble",       // SQUARE RUUBURU
	0x3355: "rem",         // SQUARE REMU
	0x3356: "roentgen",    // SQUARE RENTOGEN
	0x3357: "watt",        // SQUARE WATTO
	0x3358: "0h",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR ZERO
	0x3359: "1h",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR ONE
	0x335A: "2h",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR TWO
	0x335B: "3h",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR THREE
	0x335C: "4h",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR FOUR
	0x335D: "5h",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR FIVE
	0x335E: "6h",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR SIX
	0x335F: "7h",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR SEVEN
	0x3360: "8h",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR EIGHT
	0x3361: "9h",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR NINE
	0x3362: "10h",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR TEN
	0x3363: "11h",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR ELEVEN
	0x3364: "12h",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR TWELVE
	0x3365: "13h",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR THIRTEEN
	0x3366: "14h",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR FOURTEEN
	0x3367: "15h",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR FIFTEEN
	0x3368: "16h",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR SIXTEEN
	0x3369: "17h",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR SEVENTEEN
	0x336A: "18h",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR EIGHTEEN
	0x336B: "19h",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR NINETEEN
	0x336C: "20h",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR TWENTY
	0x336D: "21h",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR TWENTY-ONE
	0x336E: "22h",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR TWENTY-TWO
	0x336F: "23h",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR TWENTY-THREE
	0x3370: "24h",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR HOUR TWENTY-FOUR
	0x3371: "HPA",         // SQUARE HPA
	0x3372: "da",          // SQUARE DA
	0x3373: "AU",          // SQUARE AU
	0x3374: "bar",         // SQUARE BAR
	0x3375: "oV",          // SQUARE OV
	0x3376: "pc",          // SQUARE PC
	0x337B: "Heisei",      // SQUARE ERA NAME HEISEI
	0x337C: "Syouwa",      // SQUARE ERA NAME SYOUWA
	0x337D: "Taisyou",     // SQUARE ERA NAME TAISYOU
	0x337E: "Meiji",       // SQUARE ERA NAME MEIZI
	0x337F: "Inc.",        // SQUARE CORPORATION
	0x3380: "pA",          // SQUARE PA AMPS
	0x3381: "nA",          // SQUARE NA
	0x3382: "microamp",    // SQUARE MU A
	0x3383: "mA",          // SQUARE MA
	0x3384: "kA",          // SQUARE KA
	0x3385: "kB",          // SQUARE KB
	0x3386: "MB",          // SQUARE MB
	0x3387: "GB",          // SQUARE GB
	0x3388: "cal",         // SQUARE CAL
	0x3389: "kcal",        // SQUARE KCAL
	0x338A: "pF",          // SQUARE PF
	0x338B: "nF",          // SQUARE NF
	0x338C: "microFarad",  // SQUARE MU F
	0x338D: "microgram",   // SQUARE MU G
	0x338E: "mg",          // SQUARE MG
	0x338F: "kg",          // SQUARE KG
	0x3390: "Hz",          // SQUARE HZ
	0x3391: "kHz",         // SQUARE KHZ
	0x3392: "MHz",         // SQUARE MHZ
	0x3393: "GHz",         // SQUARE GHZ
	0x3394: "THz",         // SQUARE THZ
	0x3395: "microliter",  // SQUARE MU L
	0x3396: "ml",          // SQUARE ML
	0x3397: "dl",          // SQUARE DL
	0x3398: "kl",          // SQUARE KL
	0x3399: "fm",          // SQUARE FM
	0x339A: "nm",          // SQUARE NM
	0x339B: "micrometer",  // SQUARE MU M
	0x339C: "mm",          // SQUARE MM
	0x339D: "cm",          // SQUARE CM
	0x339E: "km",          // SQUARE KM
	0x339F: "mm^2",        // SQUARE MM SQUARED
	0x33A0: "cm^2",        // SQUARE CM SQUARED
	0x33A1: "m^2",         // SQUARE M SQUARED
	0x33A2: "km^2",        // SQUARE KM SQUARED
	0x33A3: "mm^4",        // SQUARE MM CUBED
	0x33A4: "cm^3",        // SQUARE CM CUBED
	0x33A5: "m^3",         // SQUARE M CUBED
	0x33A6: "km^3",        // SQUARE KM CUBED
	0x33A7: "m/s",         // SQUARE M OVER S
	0x33A8: "m/s^2",       // SQUARE M OVER S SQUARED
	0x33A9: "Pa",          // SQUARE PA
	0x33AA: "kPa",         // SQUARE KPA
	0x33AB: "MPa",         // SQUARE MPA
	0x33AC: "GPa",         // SQUARE GPA
	0x33AD: "rad",         // SQUARE RAD
	0x33AE: "rad/s",       // SQUARE RAD OVER S
	0x33AF: "rad/s^2",     // SQUARE RAD OVER S SQUARED
	0x33B0: "ps",          // SQUARE PS
	0x33B1: "ns",          // SQUARE NS
	0x33B2: "microsecond", // SQUARE MU S
	0x33B3: "ms",          // SQUARE MS
	0x33B4: "pV",          // SQUARE PV
	0x33B5: "nV",          // SQUARE NV
	0x33B6: "microvolt",   // SQUARE MU V
	0x33B7: "mV",          // SQUARE MV
	0x33B8: "kV",          // SQUARE KV
	0x33B9: "MV",          // SQUARE MV MEGA
	0x33BA: "pW",          // SQUARE PW
	0x33BB: "nW",          // SQUARE NW
	0x33BC: "microwatt",   // SQUARE MU W
	0x33BD: "mW",          // SQUARE MW
	0x33BE: "kW",          // SQUARE KW
	0x33BF: "MW",          // SQUARE MW MEGA
	0x33C0: "kOhm",        // SQUARE K OHM
	0x33C1: "MOhm",        // SQUARE M OHM
	0x33C2: "a.m.",        // SQUARE AM
	0x33C3: "Bq",          // SQUARE BQ
	0x33C4: "cc",          // SQUARE CC
	0x33C5: "cd",          // SQUARE CD
	0x33C6: "C/kg",        // SQUARE C OVER KG
	0x33C7: "Co.",         // SQUARE CO
	0x33C8: "dB",          // SQUARE DB
	0x33C9: "Gy",          // SQUARE GY
	0x33CA: "ha",          // SQUARE HA
	0x33CB: "HP",          // SQUARE HP
	0x33CC: "in",          // SQUARE IN
	0x33CD: "K.K.",        // SQUARE KK
	0x33CE: "KM",          // SQUARE KM CAPITAL
	0x33CF: "kt",          // SQUARE KT
	0x33D0: "lm",          // SQUARE LM
	0x33D1: "ln",          // SQUARE LN
	0x33D2: "log",         // SQUARE LOG
	0x33D3: "lx",          // SQUARE LX
	0x33D4: "mb",          // SQUARE MB SMALL
	0x33D5: "mil",         // SQUARE MIL
	0x33D6: "mol",         // SQUARE MOL
	0x33D7: "pH",          // SQUARE PH
	0x33D8: "p.m.",        // SQUARE PM
	0x33D9: "PPM",         // SQUARE PPM
	0x33DA: "PR",          // SQUARE PR
	0x33DB: "sr",          // SQUARE SR
	0x33DC: "Sv",          // SQUARE SV
	0x33DD: "Wb",          // SQUARE WB
	0x33E0: "1d",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY ONE
	0x33E1: "2d",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY TWO
	0x33E2: "3d",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY THREE
	0x33E3: "4d",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY FOUR
	0x33E4: "5d",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY FIVE
	0x33E5: "6d",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY SIX
	0x33E6: "7d",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY SEVEN
	0x33E7: "8d",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY EIGHT
	0x33E8: "9d",          // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY NINE
	0x33E9: "10d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY TEN
	0x33EA: "11d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY ELEVEN
	0x33EB: "12d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY TWELVE
	0x33EC: "13d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY THIRTEEN
	0x33ED: "14d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY FOURTEEN
	0x33EE: "15d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY FIFTEEN
	0x33EF: "16d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY SIXTEEN
	0x33F0: "17d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY SEVENTEEN
	0x33F1: "18d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY EIGHTEEN
	0x33F2: "19d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY NINETEEN
	0x33F3: "20d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY TWENTY
	0x33F4: "21d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY TWENTY-ONE
	0x33F5: "22d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY TWENTY-TWO
	0x33F6: "23d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY TWENTY-THREE
	0x33F7: "24d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY TWENTY-FOUR
	0x33F8: "25d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY TWENTY-FIVE
	0x33F9: "26d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY TWENTY-SIX
	0x33FA: "27d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY TWENTY-SEVEN
	0x33FB: "28d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY TWENTY-EIGHT
	0x33FC: "29d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY TWENTY-NINE
	0x33FD: "30d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY THIRTY
	0x33FE: "31d",         // IDEOGRAPHIC TELEGRAPH SYMBOL FOR DAY THIRTY-ONE
	0x33FF: "(gal)",       //
	0xFB00: "ff",          // LATIN SMALL LIGATURE FF
	0xFB01: "fi",          // LATIN SMALL LIGATURE FI
	0xFB02: "fl",          // LATIN SMALL LIGATURE FL
	0xFB03: "ffi",         // LATIN SMALL LIGATURE FFI
	0xFB04: "ffl",         // LATIN SMALL LIGATURE FFL
	0xFB05: "st",          // LATIN SMALL LIGATURE LONG S T
	0xFB06: "st",          // LATIN SMALL LIGATURE ST
	0xFB13: "mn",          // ARMENIAN SMALL LIGATURE MEN NOW
	0xFB14: "me",          // ARMENIAN SMALL LIGATURE MEN ECH
	0xFB15: "mi",          // ARMENIAN SMALL LIGATURE MEN INI
	0xFB16: "vn",          // ARMENIAN SMALL LIGATURE VEW NOW
	0xFB17: "mkh",         // ARMENIAN SMALL LIGATURE MEN XEH
	0xFB1D: "yi",          // HEBREW LETTER YOD WITH HIRIQ
	0xFB1F: "ay",          // HEBREW LIGATURE YIDDISH YOD YOD PATAH
	0xFB20: "`",           // HEBREW LETTER ALTERNATIVE AYIN
	0xFB22: "d",           // HEBREW LETTER WIDE DALET
	0xFB23: "h",           // HEBREW LETTER WIDE HE
	0xFB24: "k",           // HEBREW LETTER WIDE KAF
	0xFB25: "l",           // HEBREW LETTER WIDE LAMED
	0xFB26: "m",           // HEBREW LETTER WIDE FINAL MEM
	0xFB27: "m",           // HEBREW LETTER WIDE RESH
	0xFB28: "t",           // HEBREW LETTER WIDE TAV
	0xFB29: "+",           // HEBREW LETTER ALTERNATIVE PLUS SIGN
	0xFB2A: "sh",          // HEBREW LETTER SHIN WITH SHIN DOT
	0xFB2B: "s",           // HEBREW LETTER SHIN WITH SIN DOT
	0xFB2C: "sh",          // HEBREW LETTER SHIN WITH DAGESH AND SHIN D
	0xFB2D: "s",           // HEBREW LETTER SHIN WITH DAGESH AND SIN DO
	0xFB2E: "a",           // HEBREW LETTER ALEF WITH PATAH
	0xFB2F: "a",           // HEBREW LETTER ALEF WITH QAMATS
	0xFB31: "b",           // HEBREW LETTER BET WITH DAGESH
	0xFB32: "g",           // HEBREW LETTER GIMEL WITH DAGESH
	0xFB33: "d",           // HEBREW LETTER DALET WITH DAGESH
	0xFB34: "h",           // HEBREW LETTER HE WITH MAPIQ
	0xFB35: "v",           // HEBREW LETTER VAV WITH DAGESH
	0xFB36: "z",           // HEBREW LETTER ZAYIN WITH DAGESH
	0xFB38: "t",           // HEBREW LETTER TET WITH DAGESH
	0xFB39: "y",           // HEBREW LETTER YOD WITH DAGESH
	0xFB3A: "k",           // HEBREW LETTER FINAL KAF WITH DAGESH
	0xFB3B: "k",           // HEBREW LETTER KAF WITH DAGESH
	0xFB3C: "l",           // HEBREW LETTER LAMED WITH DAGESH
	0xFB3E: "l",           // HEBREW LETTER MEM WITH DAGESH
	0xFB40: "n",           // HEBREW LETTER NUN WITH DAGESH
	0xFB41: "n",           // HEBREW LETTER SAMEKH WITH DAGESH
	0xFB43: "p",           // HEBREW LETTER FINAL PE WITH DAGESH
	0xFB44: "p",           // HEBREW LETTER PE WITH DAGESH
	0xFB46: "ts",          // HEBREW LETTER TSADI WITH DAGESH
	0xFB47: "ts",          // HEBREW LETTER QOF WITH DAGESH
	0xFB48: "r",           // HEBREW LETTER RESH WITH DAGESH
	0xFB49: "sh",          // HEBREW LETTER SHIN WITH DAGESH
	0xFB4A: "t",           // HEBREW LETTER TAV WITH DAGESH
	0xFB4B: "vo",          // HEBREW LETTER VAV WITH HOLAM
	0xFB4C: "b",           // HEBREW LETTER BET WITH RAFE
	0xFB4D: "k",           // HEBREW LETTER KAF WITH RAFE
	0xFB4E: "p",           // HEBREW LETTER PE WITH RAFE
	0xFB4F: "l",           // HEBREW LIGATURE ALEF LAMED
	0xFE23: "~",           // COMBINING DOUBLE TILDE RIGHT HALF
	0xFE30: "..",          // PRESENTATION FORM FOR VERTICAL TWO DOT LEADER
	0xFE31: "--",          // PRESENTATION FORM FOR VERTICAL EM DASH
	0xFE32: "-",           // PRESENTATION FORM FOR VERTICAL EN DASH
	0xFE33: "_",           // PRESENTATION FORM FOR VERTICAL LOW LINE
	0xFE34: "_",           // PRESENTATION FORM FOR VERTICAL WAVY LOW LINE
	0xFE35: "(",           // PRESENTATION FORM FOR VERTICAL LEFT PARENTHESIS
	0xFE36: ") ",          // PRESENTATION FORM FOR VERTICAL RIGHT PARENTHESIS
	0xFE37: "{",           // PRESENTATION FORM FOR VERTICAL LEFT CURLY BRACKET
	0xFE38: "} ",          // PRESENTATION FORM FOR VERTICAL RIGHT CURLY BRACKET
	0xFE39: "[",           // PRESENTATION FORM FOR VERTICAL LEFT TORTOISE SHELL BRACKET
	0xFE3A: "] ",          // PRESENTATION FORM FOR VERTICAL RIGHT TORTOISE SHELL BRACKET
	0xFE3B: "[(",          // PRESENTATION FORM FOR VERTICAL LEFT BLACK LENTICULAR BRACKET
	0xFE3C: ")] ",         // PRESENTATION FORM FOR VERTICAL RIGHT BLACK LENTICULAR BRACKET
	0xFE3D: "<<",          // PRESENTATION FORM FOR VERTICAL LEFT DOUBLE ANGLE BRACKET
	0xFE3E: ">> ",         // PRESENTATION FORM FOR VERTICAL RIGHT DOUBLE ANGLE BRACKET
	0xFE3F: "<",           // PRESENTATION FORM FOR VERTICAL LEFT ANGLE BRACKET
	0xFE40: "> ",          // PRESENTATION FORM FOR VERTICAL RIGHT ANGLE BRACKET
	0xFE41: "[",           // PRESENTATION FORM FOR VERTICAL LEFT CORNER BRACKET
	0xFE42: "] ",          // PRESENTATION FORM FOR VERTICAL RIGHT CORNER BRACKET
	0xFE43: "{",           // PRESENTATION FORM FOR VERTICAL LEFT WHITE CORNER BRACKET
	0xFE44: "}",           // PRESENTATION FORM FOR VERTICAL RIGHT WHITE CORNER BRACKET
	0xFE50: ",",           // SMALL COMMA
	0xFE51: ",",           // SMALL IDEOGRAPHIC COMMA
	0xFE52: ".",           // SMALL FULL STOP
	0xFE54: ";",           // SMALL SEMICOLON
	0xFE55: ":",           // SMALL COLON
	0xFE56: "?",           // SMALL QUESTION MARK
	0xFE57: "!",           // SMALL EXCLAMATION MARK
	0xFE58: "-",           // SMALL EM DASH
	0xFE59: "(",           // SMALL LEFT PARENTHESIS
	0xFE5A: ")",           // SMALL RIGHT PARENTHESIS
	0xFE5B: "{",           // SMALL LEFT CURLY BRACKET
	0xFE5C: "}",           // SMALL RIGHT CURLY BRACKET
	0xFE5D: "{",           // SMALL LEFT TORTOISE SHELL BRACKET
	0xFE5E: "}",           // SMALL RIGHT TORTOISE SHELL BRACKET
	0xFE5F: "#",           // SMALL NUMBER SIGN
	0xFE60: "&",           // SMALL AMPERSAND
	0xFE61: "*",           // SMALL ASTERISK
	0xFE62: "+",           // SMALL PLUS SIGN
	0xFE63: "-",           // SMALL HYPHEN-MINUS
	0xFE64: "<",           // SMALL LESS-THAN SIGN
	0xFE65: ">",           // SMALL GREATER-THAN SIGN
	0xFE66: "=",           // SMALL EQUALS SIGN
	0xFE68: "\\",          // SMALL REVERSE SOLIDUS
	0xFE69: "$",           // SMALL DOLLAR SIGN
	0xFE6A: "%",           // SMALL PERCENT SIGN
	0xFE6B: "@",           // SMALL COMMERCIAL AT
	0xFF01: "!",           // FULLWIDTH EXCLAMATION MARK
	0xFF02: "\"",          // FULLWIDTH QUOTATION MARK
	0xFF03: "#",           // FULLWIDTH NUMBER SIGN
	0xFF04: "$",           // FULLWIDTH DOLLAR SIGN
	0xFF05: "%",           // FULLWIDTH PERCENT SIGN
	0xFF06: "&",           // FULLWIDTH AMPERSAND
	0xFF07: "'",           // FULLWIDTH APOSTROPHE
	0xFF08: "(",           // FULLWIDTH LEFT PARENTHESIS
	0xFF09: ")",           // FULLWIDTH RIGHT PARENTHESIS
	0xFF0A: "*",           // FULLWIDTH ASTERISK
	0xFF0B: "+",           // FULLWIDTH PLUS SIGN
	0xFF0C: ",",           // FULLWIDTH COMMA
	0xFF0D: "-",           // FULLWIDTH HYPHEN-MINUS
	0xFF0E: ".",           // FULLWIDTH FULL STOP
	0xFF0F: "/",           // FULLWIDTH SOLIDUS
	0xFF10: "0",           // FULLWIDTH DIGIT ZERO
	0xFF11: "1",           // FULLWIDTH DIGIT ONE
	0xFF12: "2",           // FULLWIDTH DIGIT TWO
	0xFF13: "3",           // FULLWIDTH DIGIT THREE
	0xFF14: "4",           // FULLWIDTH DIGIT FOUR
	0xFF15: "5",           // FULLWIDTH DIGIT FIVE
	0xFF16: "6",           // FULLWIDTH DIGIT SIX
	0xFF17: "7",           // FULLWIDTH DIGIT SEVEN
	0xFF18: "8",           // FULLWIDTH DIGIT EIGHT
	0xFF19: "9",           // FULLWIDTH DIGIT NINE
	0xFF1A: ":",           // FULLWIDTH COLON
	0xFF1B: ";",           // FULLWIDTH SEMICOLON
	0xFF1C: "<",           // FULLWIDTH LESS-THAN SIGN
	0xFF1D: "=",           // FULLWIDTH EQUALS SIGN
	0xFF1E: ">",           // FULLWIDTH GREATER-THAN SIGN
	0xFF1F: "?",           // FULLWIDTH QUESTION MARK
	0xFF20: "@",           // FULLWIDTH COMMERCIAL AT
	0xFF21: "A",           // FULLWIDTH LATIN CAPITAL LETTER A
	0xFF22: "B",           // FULLWIDTH LATIN CAPITAL LETTER B
	0xFF23: "C",           // FULLWIDTH LATIN CAPITAL LETTER C
	0xFF24: "D",           // FULLWIDTH LATIN CAPITAL LETTER D
	0xFF25: "E",           // FULLWIDTH LATIN CAPITAL LETTER E
	0xFF26: "F",           // FULLWIDTH LATIN CAPITAL LETTER F
	0xFF27: "G",           // FULLWIDTH LATIN CAPITAL LETTER G
	0xFF28: "H",           // FULLWIDTH LATIN CAPITAL LETTER H
	0xFF29: "I",           // FULLWIDTH LATIN CAPITAL LETTER I
	0xFF2A: "J",           // FULLWIDTH LATIN CAPITAL LETTER J
	0xFF2B: "K",           // FULLWIDTH LATIN CAPITAL LETTER K
	0xFF2C: "L",           // FULLWIDTH LATIN CAPITAL LETTER L
	0xFF2D: "M",           // FULLWIDTH LATIN CAPITAL LETTER M
	0xFF2E: "N",           // FULLWIDTH LATIN CAPITAL LETTER N
	0xFF2F: "O",           // FULLWIDTH LATIN CAPITAL LETTER O
	0xFF30: "P",           // FULLWIDTH LATIN CAPITAL LETTER P
	0xFF31: "Q",           // FULLWIDTH LATIN CAPITAL LETTER Q
	0xFF32: "R",           // FULLWIDTH LATIN CAPITAL LETTER R
	0xFF33: "S",           // FULLWIDTH LATIN CAPITAL LETTER S
	0xFF34: "T",           // FULLWIDTH LATIN CAPITAL LETTER T
	0xFF35: "U",           // FULLWIDTH LATIN CAPITAL LETTER U
	0xFF36: "V",           // FULLWIDTH LATIN CAPITAL LETTER V
	0xFF37: "W",           // FULLWIDTH LATIN CAPITAL LETTER W
	0xFF38: "X",           // FULLWIDTH LATIN CAPITAL LETTER X
	0xFF39: "Y",           // FULLWIDTH LATIN CAPITAL LETTER Y
	0xFF3A: "Z",           // FULLWIDTH LATIN CAPITAL LETTER Z
	0xFF3B: "[",           // FULLWIDTH LEFT SQUARE BRACKET
	0xFF3C: "\\",          // FULLWIDTH REVERSE SOLIDUS
	0xFF3D: "]",           // FULLWIDTH RIGHT SQUARE BRACKET
	0xFF3E: "^",           // FULLWIDTH CIRCUMFLEX ACCENT
	0xFF3F: "_",           // FULLWIDTH LOW LINE
	0xFF40: "`",           // FULLWIDTH GRAVE ACCENT
	0xFF41: "a",           // FULLWIDTH LATIN SMALL LETTER A
	0xFF42: "b",           // FULLWIDTH LATIN SMALL LETTER B
	0xFF43: "c",           // FULLWIDTH LATIN SMALL LETTER C
	0xFF44: "d",           // FULLWIDTH LATIN SMALL LETTER D
	0xFF45: "e",           // FULLWIDTH LATIN SMALL LETTER E
	0xFF46: "f",           // FULLWIDTH LATIN SMALL LETTER F
	0xFF47: "g",           // FULLWIDTH LATIN SMALL LETTER G
	0xFF48: "h",           // FULLWIDTH LATIN SMALL LETTER H
	0xFF49: "i",           // FULLWIDTH LATIN SMALL LETTER I
	0xFF4A: "j",           // FULLWIDTH LATIN SMALL LETTER J
	0xFF4B: "k",           // FULLWIDTH LATIN SMALL LETTER K
	0xFF4C: "l",           // FULLWIDTH LATIN SMALL LETTER L
	0xFF4D: "m",           // FULLWIDTH LATIN SMALL LETTER M
	0xFF4E: "n",           // FULLWIDTH LATIN SMALL LETTER N
	0xFF4F: "o",           // FULLWIDTH LATIN SMALL LETTER O
	0xFF50: "p",           // FULLWIDTH LATIN SMALL LETTER P
	0xFF51: "q",           // FULLWIDTH LATIN SMALL LETTER Q
	0xFF52: "r",           // FULLWIDTH LATIN SMALL LETTER R
	0xFF53: "s",           // FULLWIDTH LATIN SMALL LETTER S
	0xFF54: "t",           // FULLWIDTH LATIN SMALL LETTER T
	0xFF55: "u",           // FULLWIDTH LATIN SMALL LETTER U
	0xFF56: "v",           // FULLWIDTH LATIN SMALL LETTER V
	0xFF57: "w",           // FULLWIDTH LATIN SMALL LETTER W
	0xFF58: "x",           // FULLWIDTH LATIN SMALL LETTER X
	0xFF59: "y",           // FULLWIDTH LATIN SMALL LETTER Y
	0xFF5A: "z",           // FULLWIDTH LATIN SMALL LETTER Z
	0xFF5B: "{",           // FULLWIDTH LEFT CURLY BRACKET
	0xFF5C: "|",           // FULLWIDTH VERTICAL LINE
	0xFF5D: "}",           // FULLWIDTH RIGHT CURLY BRACKET
	0xFF5E: "~",           // FULLWIDTH TILDE
	0xFF5F: "",            //
	0xFF60: "",            //
	0xFF61: ".",           // HALFWIDTH IDEOGRAPHIC FULL STOP
	0xFF62: "[",           // HALFWIDTH LEFT CORNER BRACKET
	0xFF63: "]",           // HALFWIDTH RIGHT CORNER BRACKET
	0xFF64: ",",           // HALFWIDTH IDEOGRAPHIC COMMA
	0xFF65: "*",           // HALFWIDTH KATAKANA MIDDLE DOT
	0xFF66: "wo",          // HALFWIDTH KATAKANA LETTER WO
	0xFF67: "a",           // HALFWIDTH KATAKANA LETTER SMALL A
	0xFF68: "i",           // HALFWIDTH KATAKANA LETTER SMALL I
	0xFF69: "u",           // HALFWIDTH KATAKANA LETTER SMALL U
	0xFF6A: "e",           // HALFWIDTH KATAKANA LETTER SMALL E
	0xFF6B: "o",           // HALFWIDTH KATAKANA LETTER SMALL O
	0xFF6C: "ya",          // HALFWIDTH KATAKANA LETTER SMALL YA
	0xFF6D: "yu",          // HALFWIDTH KATAKANA LETTER SMALL YU
	0xFF6E: "yo",          // HALFWIDTH KATAKANA LETTER SMALL YO
	0xFF6F: "tu",          // HALFWIDTH KATAKANA LETTER SMALL TU
	0xFF70: "+",           // HALFWIDTH KATAKANA-HIRAGANA PROLONGED SOUND MARK
	0xFF71: "a",           // HALFWIDTH KATAKANA LETTER A
	0xFF72: "i",           // HALFWIDTH KATAKANA LETTER I
	0xFF73: "u",           // HALFWIDTH KATAKANA LETTER U
	0xFF74: "e",           // HALFWIDTH KATAKANA LETTER E
	0xFF75: "o",           // HALFWIDTH KATAKANA LETTER O
	0xFF76: "ka",          // HALFWIDTH KATAKANA LETTER KA
	0xFF77: "ki",          // HALFWIDTH KATAKANA LETTER KI
	0xFF78: "ku",          // HALFWIDTH KATAKANA LETTER KU
	0xFF79: "ke",          // HALFWIDTH KATAKANA LETTER KE
	0xFF7A: "ko",          // HALFWIDTH KATAKANA LETTER KO
	0xFF7B: "sa",          // HALFWIDTH KATAKANA LETTER SA
	0xFF7C: "si",          // HALFWIDTH KATAKANA LETTER SI
	0xFF7D: "su",          // HALFWIDTH KATAKANA LETTER SU
	0xFF7E: "se",          // HALFWIDTH KATAKANA LETTER SE
	0xFF7F: "so",          // HALFWIDTH KATAKANA LETTER SO
	0xFF80: "ta",          // HALFWIDTH KATAKANA LETTER TA
	0xFF81: "ti",          // HALFWIDTH KATAKANA LETTER TI
	0xFF82: "tu",          // HALFWIDTH KATAKANA LETTER TU
	0xFF83: "te",          // HALFWIDTH KATAKANA LETTER TE
	0xFF84: "to",          // HALFWIDTH KATAKANA LETTER TO
	0xFF85: "na",          // HALFWIDTH KATAKANA LETTER NA
	0xFF86: "ni",          // HALFWIDTH KATAKANA LETTER NI
	0xFF87: "nu",          // HALFWIDTH KATAKANA LETTER NU
	0xFF88: "ne",          // HALFWIDTH KATAKANA LETTER NE
	0xFF89: "no",          // HALFWIDTH KATAKANA LETTER NO
	0xFF8A: "ha",          // HALFWIDTH KATAKANA LETTER HA
	0xFF8B: "hi",          // HALFWIDTH KATAKANA LETTER HI
	0xFF8C: "hu",          // HALFWIDTH KATAKANA LETTER HU
	0xFF8D: "he",          // HALFWIDTH KATAKANA LETTER HE
	0xFF8E: "ho",          // HALFWIDTH KATAKANA LETTER HO
	0xFF8F: "ma",          // HALFWIDTH KATAKANA LETTER MA
	0xFF90: "mi",          // HALFWIDTH KATAKANA LETTER MI
	0xFF91: "mu",          // HALFWIDTH KATAKANA LETTER MU
	0xFF92: "me",          // HALFWIDTH KATAKANA LETTER ME
	0xFF93: "mo",          // HALFWIDTH KATAKANA LETTER MO
	0xFF94: "ya",          // HALFWIDTH KATAKANA LETTER YA
	0xFF95: "yu",          // HALFWIDTH KATAKANA LETTER YU
	0xFF96: "yo",          // HALFWIDTH KATAKANA LETTER YO
	0xFF97: "ra",          // HALFWIDTH KATAKANA LETTER RA
	0xFF98: "ri",          // HALFWIDTH KATAKANA LETTER RI
	0xFF99: "ru",          // HALFWIDTH KATAKANA LETTER RU
	0xFF9A: "re",          // HALFWIDTH KATAKANA LETTER RE
	0xFF9B: "ro",          // HALFWIDTH KATAKANA LETTER RO
	0xFF9C: "wa",          // HALFWIDTH KATAKANA LETTER WA
	0xFF9D: "n",           // HALFWIDTH KATAKANA LETTER N
	0xFF9E: ":",           // HALFWIDTH KATAKANA VOICED SOUND MARK
	0xFF9F: ";",           // HALFWIDTH KATAKANA SEMI-VOICED SOUND MARK
	0xFFA0: "",            // HALFWIDTH HANGUL FILLER
	0xFFA1: "g",           // HALFWIDTH HANGUL LETTER KIYEOK
	0xFFA2: "gg",          // HALFWIDTH HANGUL LETTER SSANGKIYEOK
	0xFFA3: "gs",          // HALFWIDTH HANGUL LETTER KIYEOK-SIOS
	0xFFA4: "n",           // HALFWIDTH HANGUL LETTER NIEUN
	0xFFA5: "nj",          // HALFWIDTH HANGUL LETTER NIEUN-CIEUC
	0xFFA6: "nh",          // HALFWIDTH HANGUL LETTER NIEUN-HIEUH
	0xFFA7: "d",           // HALFWIDTH HANGUL LETTER TIKEUT
	0xFFA8: "dd",          // HALFWIDTH HANGUL LETTER SSANGTIKEUT
	0xFFA9: "r",           // HALFWIDTH HANGUL LETTER RIEUL
	0xFFAA: "lg",          // HALFWIDTH HANGUL LETTER RIEUL-KIYEOK
	0xFFAB: "lm",          // HALFWIDTH HANGUL LETTER RIEUL-MIEUM
	0xFFAC: "lb",          // HALFWIDTH HANGUL LETTER RIEUL-PIEUP
	0xFFAD: "ls",          // HALFWIDTH HANGUL LETTER RIEUL-SIOS
	0xFFAE: "lt",          // HALFWIDTH HANGUL LETTER RIEUL-THIEUTH
	0xFFAF: "lp",          // HALFWIDTH HANGUL LETTER RIEUL-PHIEUPH
	0xFFB0: "rh",          // HALFWIDTH HANGUL LETTER RIEUL-HIEUH
	0xFFB1: "m",           // HALFWIDTH HANGUL LETTER MIEUM
	0xFFB2: "b",           // HALFWIDTH HANGUL LETTER PIEUP
	0xFFB3: "bb",          // HALFWIDTH HANGUL LETTER SSANGPIEUP
	0xFFB4: "bs",          // HALFWIDTH HANGUL LETTER PIEUP-SIOS
	0xFFB5: "s",           // HALFWIDTH HANGUL LETTER SIOS
	0xFFB6: "ss",          // HALFWIDTH HANGUL LETTER SSANGSIOS
	0xFFB7: "",            // HALFWIDTH HANGUL LETTER IEUNG
	0xFFB8: "j",           // HALFWIDTH HANGUL LETTER CIEUC
	0xFFB9: "jj",          // HALFWIDTH HANGUL LETTER SSANGCIEUC
	0xFFBA: "c",           // HALFWIDTH HANGUL LETTER CHIEUCH
	0xFFBB: "k",           // HALFWIDTH HANGUL LETTER KHIEUKH
	0xFFBC: "t",           // HALFWIDTH HANGUL LETTER THIEUTH
	0xFFBD: "p",           // HALFWIDTH HANGUL LETTER PHIEUPH
	0xFFBE: "h",           // HALFWIDTH HANGUL LETTER HIEUH
	0xFFC2: "a",           // HALFWIDTH HANGUL LETTER A
	0xFFC3: "ae",          // HALFWIDTH HANGUL LETTER AE
	0xFFC4: "ya",          // HALFWIDTH HANGUL LETTER YA
	0xFFC5: "yae",         // HALFWIDTH HANGUL LETTER YAE
	0xFFC6: "eo",          // HALFWIDTH HANGUL LETTER EO
	0xFFC7: "e",           // HALFWIDTH HANGUL LETTER E
	0xFFCA: "yeo",         // HALFWIDTH HANGUL LETTER YEO
	0xFFCB: "ye",          // HALFWIDTH HANGUL LETTER YE
	0xFFCC: "o",           // HALFWIDTH HANGUL LETTER O
	0xFFCD: "wa",          // HALFWIDTH HANGUL LETTER WA
	0xFFCE: "wae",         // HALFWIDTH HANGUL LETTER WAE
	0xFFCF: "oe",          // HALFWIDTH HANGUL LETTER OE
	0xFFD2: "yo",          // HALFWIDTH HANGUL LETTER YO
	0xFFD3: "u",           // HALFWIDTH HANGUL LETTER U
	0xFFD4: "weo",         // HALFWIDTH HANGUL LETTER WEO
	0xFFD5: "we",          // HALFWIDTH HANGUL LETTER WE
	0xFFD6: "wi",          // HALFWIDTH HANGUL LETTER WI
	0xFFD7: "yu",          // HALFWIDTH HANGUL LETTER YU
	0xFFDA: "eu",          // HALFWIDTH HANGUL LETTER EU
	0xFFDB: "yi",          // HALFWIDTH HANGUL LETTER YI
	0xFFDC: "i",           // HALFWIDTH HANGUL LETTER I
	0xFFE0: "/C",          // FULLWIDTH CENT SIGN
	0xFFE1: "PS",          // FULLWIDTH POUND SIGN
	0xFFE2: "!",           // FULLWIDTH NOT SIGN
	0xFFE3: "-",           // FULLWIDTH MACRON
	0xFFE4: "|",           // FULLWIDTH BROKEN BAR
	0xFFE5: "Y=",          // FULLWIDTH YEN SIGN
	0xFFE6: "W=",          // FULLWIDTH WON SIGN
	0xFFE8: "|",           // HALFWIDTH FORMS LIGHT VERTICAL
	0xFFE9: "-",           // HALFWIDTH LEFTWARDS ARROW
	0xFFEA: "|",           // HALFWIDTH UPWARDS ARROW
	0xFFEB: "-",           // HALFWIDTH RIGHTWARDS ARROW
	0xFFEC: "|",           // HALFWIDTH DOWNWARDS ARROW
	0xFFED: "#",           // HALFWIDTH BLACK SQUARE
	0xFFEE: "O",           // HALFWIDTH WHITE CIRCLE
	0xFFF9: "{",           // INTERLINEAR ANNOTATION ANCHOR
	0xFFFA: "|",           // INTERLINEAR ANNOTATION SEPARATOR
	0xFFFB: "}",           // INTERLINEAR ANNOTATION TERMINATOR
	0xFFFD: "?",           // REPLACEMENT CHARACTER
}

// FixMisusedLetters fixes Greek beta and German sharp s misuse
func FixMisusedLetters(str string, doHomoglyphs, isAuthor, isProse bool) string {

	var arry []string

	isReallyEszett := func(before, after string, mustBeAllCaps bool) bool {

		// trim at closest space before sharp s rune
		idx := strings.LastIndex(before, " ")
		if idx >= 0 {
			before = before[idx+1:]
		}

		// trim at closest space after sharp s rune
		idx = strings.Index(after, " ")
		if idx >= 0 {
			after = after[:idx]
		}

		isAllCaps := true

		// look at characters to the left
		if before == "" {
			// must not be first character
			return false
		}
		last := rune(0)
		for _, ch := range before {
			_, ok := germanRunes[ch]
			if !ok {
				return false
			}
			_, ok = germanCapitals[ch]
			if !ok {
				isAllCaps = false
			}
			last = ch
		}
		_, ok := germanVowels[last]
		if !ok {
			// character immediately to the left must be a vowel (actually, a long vowel or diphthong)
			// though after a consonant, if it really is German, it should become ss
			// perhaps that situation can be added to a future table for handling specific exceptions
			return false
		}

		// look at characters to the right
		for _, ch := range after {
			_, ok := germanRunes[ch]
			if !ok {
				return false
			}
			_, ok = germanCapitals[ch]
			if !ok {
				isAllCaps = false
			}
			last = ch
		}

		if mustBeAllCaps {
			// capital sharp S expects all letters to be capitalized
			if !isAllCaps {
				return false
			}
		} else if isAllCaps {
			// otherwise may be a gene name, should be using beta
			return false
		}

		return true
	}

	for i, ch := range str {

		if ch < 32 {
			// skip ASCII control characters
			continue
		}

		if doHomoglyphs {
			if ch == 976 || ch == 7517 || ch == 7526 {
				// convert curled (U+03D0), modifier (U+1D5D), and subscript (U+1D66) lookalikes
				// to lower-case Greek beta U+03B2)
				ch = 946
			} else if ch == 400 {
				// capital Latin open E (U+0190) to capital Greek Epsilon (U+0395)
				ch = 917
			} else if ch == 603 {
				// lower-case Latin open E (U+025B) to lower-case Greek epsilon (U+03B5)
				ch = 949
			}
		}

		if isAuthor {
			if ch == 946 || ch == 976 || ch == 7517 || ch == 7526 {
				// in author, replace lower-case Greek beta (U+03B2) with German sharp s (U+00DF),
				// also handles curled (U+03D0), modifier (U+1D5D), and subscript (U+1D66) lookalikes
				ch = 223
			} else if ch == 914 {
				// replace upper-case Greek Beta (U+0392) with Latin capital B
				ch = 66
			} else if ch == 34 {
				// and replace double quote by apostrophe
				ch = 39
			}
		}

		if isProse {
			if ch == 223 {
				// in text, German sharp s is occasionally correct
				if !isReallyEszett(str[:i], str[i+utf8.RuneLen(ch):], false) {
					// but in scientific papers usually should be Greek beta
					ch = 946
				}
			} else if ch == 946 || ch == 976 || ch == 7517 || ch == 7526 {
				// sometimes Greek beta should actually be German sharp s
				if isReallyEszett(str[:i], str[i+utf8.RuneLen(ch):], false) {
					ch = 223
				}
			} else if ch == 7838 {
				// also check whether to convert capitalized German sharp S (U+1E9E)
				if !isReallyEszett(str[:i], str[i+utf8.RuneLen(ch):], true) {
					ch = 946
				}
			}
		}

		arry = append(arry, string(ch))
	}

	str = strings.Join(arry, "")

	return str
}

// mutex to protect loading of external Unicode transformation table
var ulock sync.Mutex

// TransformAccents converts accented letters and symbols to closest ASCII equivalent
func TransformAccents(str string, spellGreek, reEncode bool) string {

	var arry []string

	// CJK (Chinese, Japanese, and Korean) Unified Ideograph Extensions, and Private Use Surrogates
	loadExternRunes := func() bool {

		loaded := false

		ex, eerr := os.Executable()
		if eerr != nil {
			return false
		}

		exPath := filepath.Dir(ex)
		fpath := filepath.Join(exPath, "help", "unicode-extras.txt")
		file, ferr := os.Open(fpath)

		if file != nil && ferr == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				str := scanner.Text()
				if str == "" {
					continue
				}
				cols := strings.SplitN(str, "\t", 2)
				if len(cols) != 2 {
					continue
				}
				n, err := strconv.ParseUint(cols[0], 16, 32)
				if err != nil {
					continue
				}
				ch := rune(n)
				st := cols[1]
				externRunes[ch] = st
				loaded = true
			}
		}
		file.Close()

		return loaded
	}

	for _, ch := range str {
		st := ""
		ok := false

		if ch < 128 {
			// add printable 7-bit ASCII character directly
			if ch > 31 {
				arry = append(arry, string(ch))
			}
			continue
		}

		if spellGreek {
			// spells Greek letters (e.g., alpha, beta) for easier searching,
			// handles glyph variants, treats Latin letter open E as Greek epsilon
			st, ok := greekRunes[ch]
			if ok {
				arry = append(arry, st)
				continue
			}
		}

		// lookup remaining characters in asciiRunes table
		st, ok = asciiRunes[ch]
		if !ok {
			// try symbolRunes table next
			st, ok = symbolRunes[ch]
		}
		if !ok {
			st, ok = extraRunes[ch]
		}
		if !ok && ch >= 0x0300 && ch <= 0x036F {
			// absorb combining accents
			continue
		}
		if !ok && ch > 0x33FF && ch < 0xFB00 {
			// load external table within mutex
			ulock.Lock()
			if !extRunesLoaded {
				extRunesLoaded = loadExternRunes()
			}
			ulock.Unlock()
			// may also check external ideograms
			if extRunesLoaded {
				st, ok = externRunes[ch]
			}
		}

		if ok {
			// leading and trailing spaces, if needed, are in maps
			arry = append(arry, st)
		}
	}

	str = strings.Join(arry, "")

	if reEncode {
		// reencode angle brackets and ampersand in XML
		str = encodeAngleBracketsAndAmpersand(str)
	}

	return str
}

// encodeAngleBracketsAndAmpersand reencodes angle brackets and ampersand in XML
func encodeAngleBracketsAndAmpersand(str string) string {

	if rfix != nil {
		for _, ch := range str {
			// check for presence of angle brackets or ampersand
			if ch == '<' || ch == '>' || ch == '&' {
				// replace all occurrences and return result
				str = rfix.Replace(str)
				return str
			}
		}
	}

	return str
}

// initialize rfix replacer and empty externRunes map before non-init functions are called
func init() {

	rfix = strings.NewReplacer(
		"<", "&lt;",
		">", "&gt;",
		"&", "&amp;",
	)

	externRunes = make(map[rune]string)
}
