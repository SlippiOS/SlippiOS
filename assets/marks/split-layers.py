#!/usr/bin/env python3
"""Split the SlippiOS mark into the pulsing shine and the static S layers.

The S is the only path with a 2.5 stroke; every other path is a shine facet.
"""

import sys
import xml.etree.ElementTree as ET


def main() -> None:
    if len(sys.argv) != 4:
        raise SystemExit("usage: split-layers.py SHINE_OUT S_OUT SOURCE")

    shine_out, s_out, src = sys.argv[1:]
    ET.register_namespace("", "http://www.w3.org/2000/svg")
    ET.register_namespace("svg", "http://www.w3.org/2000/svg")

    for out, keep_s in ((shine_out, False), (s_out, True)):
        tree = ET.parse(src)
        root = tree.getroot()
        paths = [e for e in root.iter() if e.tag.split("}")[-1] == "path"]
        parent = {child: node for node in root.iter() for child in node}
        s_paths = [
            path for path in paths
            if "stroke-width:2.5" in (path.get("style") or "")
        ]
        for path in paths:
            if (path in s_paths) != keep_s:
                parent[path].remove(path)
        tree.write(out, encoding="utf-8", xml_declaration=True)


if __name__ == "__main__":
    main()
