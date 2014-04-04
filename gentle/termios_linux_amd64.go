// Contents of this file is a direct port of termios-related functionaly from musl libc, http://www.musl-libc.org/
// musl as a whole is licensed under the following standard MIT license:
//
// ----------------------------------------------------------------------
// Copyright © 2005-2014 Rich Felker, et al.
//
// 	Permission is hereby granted, free of charge, to any person obtaining
// a copy of this software and associated documentation files (the
// 	"Software"), to deal in the Software without restriction, including
// without limitation the rights to use, copy, modify, merge, publish,
// distribute, sublicense, and/or sell copies of the Software, and to
// permit persons to whom the Software is furnished to do so, subject to
// the following conditions:
//
// The above copyright notice and this permission notice shall be
// included in all copies or substantial portions of the Software.
//
// 	THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND,
// EXPRESS OR IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF
// MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT.
// IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY
// CLAIM, DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT,
// TORT OR OTHERWISE, ARISING FROM, OUT OF OR IN CONNECTION WITH THE
// SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.
// ----------------------------------------------------------------------
//
// Authors/contributors include:
//
// Anthony G. Basile
// Arvid Picciani
// Bobby Bingham
// Boris Brezillon
// Chris Spiegel
// Emil Renner Berthing
// Hiltjo Posthuma
// Isaac Dunham
// Jens Gustedt
// Jeremy Huntwork
// John Spencer
// Justin Cormack
// Luca Barbato
// Luka Perkov
// Michael Forney
// Nicholas J. Kain
// orc
// Pascal Cuoq
// Pierre Carrier
// Rich Felker
// Richard Pennington
// Solar Designer
// Strake
// Szabolcs Nagy
// Timo Teräs
// Valentin Ochs
// William Haddon
//
// Portions of this software are derived from third-party works licensed
// under terms compatible with the above MIT license:
//
// The TRE regular expression implementation (src/regex/reg* and
// 	src/regex/tre*) is Copyright © 2001-2008 Ville Laurikari and licensed
// under a 2-clause BSD license (license text in the source files). The
// included version has been heavily modified by Rich Felker in 2012, in
// the interests of size, simplicity, and namespace cleanliness.
//
// 	Much of the math library code (src/math/* and src/complex/*) is
// Copyright © 1993,2004 Sun Microsystems or
// Copyright © 2003-2011 David Schultz or
// Copyright © 2003-2009 Steven G. Kargl or
// Copyright © 2003-2009 Bruce D. Evans or
// Copyright © 2008 Stephen L. Moshier
// and labelled as such in comments in the individual source files. All
// have been licensed under extremely permissive terms.
//
// The ARM memcpy code (src/string/armel/memcpy.s) is Copyright © 2008
// The Android Open Source Project and is licensed under a two-clause BSD
// license. It was taken from Bionic libc, used on Android.
//
// The implementation of DES for crypt (src/misc/crypt_des.c) is
// Copyright © 1994 David Burren. It is licensed under a BSD license.
//
// The implementation of blowfish crypt (src/misc/crypt_blowfish.c) was
// originally written by Solar Designer and placed into the public
// domain. The code also comes with a fallback permissive license for use
// in jurisdictions that may not recognize the public domain.
//
// The smoothsort implementation (src/stdlib/qsort.c) is Copyright © 2011
// Valentin Ochs and is licensed under an MIT-style license.
//
// The BSD PRNG implementation (src/prng/random.c) and XSI search API
// (src/search/*.c) functions are Copyright © 2011 Szabolcs Nagy and
// licensed under following terms: "Permission to use, copy, modify,
// and/or distribute this code for any purpose with or without fee is
// hereby granted. There is no warranty."
//
// The x86_64 port was written by Nicholas J. Kain. Several files (crt)
// were released into the public domain; others are licensed under the
// standard MIT license terms at the top of this file. See individual
// files for their copyright status.
//
// The mips and microblaze ports were originally written by Richard
// Pennington for use in the ellcc project. The original code was adapted
// by Rich Felker for build system and code conventions during upstream
// integration. It is licensed under the standard MIT terms.
//
// The powerpc port was also originally written by Richard Pennington,
// and later supplemented and integrated by John Spencer. It is licensed
// under the standard MIT terms.
//
// All other files which have no copyright comments are original works
// produced specifically for use as part of this library, written either
// by Rich Felker, the main author of the library, or by one or more
// contibutors listed above. Details on authorship of individual files
// can be found in the git version control history of the project. The
// omission of copyright and license comments in each file is in the
// interest of source tree size.
//
// All public header files (include/* and arch/*/bits/*) should be
// treated as Public Domain as they intentionally contain no content
// which can be covered by copyright. Some source modules may fall in
// this category as well. If you believe that a file is so trivial that
// it should be in the Public Domain, please contact the authors and
// request an explicit statement releasing it from copyright.
//
// The following files are trivial, believed not to be copyrightable in
// the first place, and hereby explicitly released to the Public Domain:
//
// All public headers: include/*, arch/*/bits/*
// Startup files: crt/*
package main

type termios struct {
	c_iflag  flag_t
	c_oflag  flag_t
	c_cflag  flag_t
	c_lflag  flag_t
	c_line   byte
	c_cc     [NCCS]byte
	c_ispeed speed_t
	c_ospeed speed_t
}

const (
	VINTR    = 0
	VQUIT    = 1
	VERASE   = 2
	VKILL    = 3
	VEOF     = 4
	VTIME    = 5
	VMIN     = 6
	VSWTC    = 7
	VSTART   = 8
	VSTOP    = 9
	VSUSP    = 10
	VEOL     = 11
	VREPRINT = 12
	VDISCARD = 13
	VWERASE  = 14
	VLNEXT   = 15
	VEOL2    = 16

	IGNBRK  = 0000001
	BRKINT  = 0000002
	IGNPAR  = 0000004
	PARMRK  = 0000010
	INPCK   = 0000020
	ISTRIP  = 0000040
	INLCR   = 0000100
	IGNCR   = 0000200
	ICRNL   = 0000400
	IUCLC   = 0001000
	IXON    = 0002000
	IXANY   = 0004000
	IXOFF   = 0010000
	IMAXBEL = 0020000
	IUTF8   = 0040000

	OPOST  = 0000001
	OLCUC  = 0000002
	ONLCR  = 0000004
	OCRNL  = 0000010
	ONOCR  = 0000020
	ONLRET = 0000040
	OFILL  = 0000100
	OFDEL  = 0000200
	NLDLY  = 0000400
	NL0    = 0000000
	NL1    = 0000400
	CRDLY  = 0003000
	CR0    = 0000000
	CR1    = 0001000
	CR2    = 0002000
	CR3    = 0003000
	TABDLY = 0014000
	TAB0   = 0000000
	TAB1   = 0004000
	TAB2   = 0010000
	TAB3   = 0014000
	BSDLY  = 0020000
	BS0    = 0000000
	BS1    = 0020000
	FFDLY  = 0100000
	FF0    = 0000000
	FF1    = 0100000

	VTDLY = 0040000
	VT0   = 0000000
	VT1   = 0040000

	B0     = 0000000
	B50    = 0000001
	B75    = 0000002
	B110   = 0000003
	B134   = 0000004
	B150   = 0000005
	B200   = 0000006
	B300   = 0000007
	B600   = 0000010
	B1200  = 0000011
	B1800  = 0000012
	B2400  = 0000013
	B4800  = 0000014
	B9600  = 0000015
	B19200 = 0000016
	B38400 = 0000017

	B57600   = 0010001
	B115200  = 0010002
	B230400  = 0010003
	B460800  = 0010004
	B500000  = 0010005
	B576000  = 0010006
	B921600  = 0010007
	B1000000 = 0010010
	B1152000 = 0010011
	B1500000 = 0010012
	B2000000 = 0010013
	B2500000 = 0010014
	B3000000 = 0010015
	B3500000 = 0010016
	B4000000 = 0010017

	CBAUD = 0010017

	CSIZE  = 0000060
	CS5    = 0000000
	CS6    = 0000020
	CS7    = 0000040
	CS8    = 0000060
	CSTOPB = 0000100
	CREAD  = 0000200
	PARENB = 0000400
	PARODD = 0001000
	HUPCL  = 0002000
	CLOCAL = 0004000

	ISIG   = 0000001
	ICANON = 0000002
	ECHO   = 0000010
	ECHOE  = 0000020
	ECHOK  = 0000040
	ECHONL = 0000100
	NOFLSH = 0000200
	TOSTOP = 0000400
	IEXTEN = 0100000

	ECHOCTL = 0001000
	ECHOPRT = 0002000
	ECHOKE  = 0004000
	FLUSHO  = 0010000
	PENDIN  = 0040000

	TCOOFF = 0
	TCOON  = 1
	TCIOFF = 2
	TCION  = 3

	TCIFLUSH  = 0
	TCOFLUSH  = 1
	TCIOFLUSH = 2

	TCSANOW   = 0
	TCSADRAIN = 1
	TCSAFLUSH = 2

	CBAUDEX = 0010000
	CRTSCTS = 020000000000
	EXTPROC = 0200000
	XTABS   = 0014000
)
