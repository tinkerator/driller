# driller - a go tool to unroll .drl files

## Introduction

Driller takes as input a `.PTH.drl` file from KiCad and generates N files:

- `.t<diameter>.PTH.drl` files, one for each of the tools
- `.drl.svg` displaying the flattened and unrolled image of the holes

## TODO

- Generate SVG output
- Generate Snapmaker 350 cnc files for the different tools

## License

The `driller` package and examples are distributed with the same BSD
3-clause [license](LICENSE) as that used by
[golang](https://golang.org/LICENSE) itself.

## Requesting features and reporting bugs

This is a hobby project. No support should be expected. However, if
you want to suggest a feature, or if you find a bug, please use the
github [driller bug
tracker](https://github.com/tinkerator/driller/issues).
