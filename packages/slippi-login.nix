{ lib, python3, webkitgtk_4_1, gtk3, gdk-pixbuf, glib, writeShellScriptBin }:

let
  python = python3.withPackages (ps: [ ps.pygobject3 ]);
in writeShellScriptBin "slippi-login" ''
  export GI_TYPELIB_PATH=${webkitgtk_4_1}/lib/girepository-1.0:${gtk3}/lib/girepository-1.0:${gdk-pixbuf}/lib/girepository-1.0:${glib}/lib/girepository-1.0
  export LD_LIBRARY_PATH=${webkitgtk_4_1}/lib:${gtk3}/lib
  export XDG_DATA_DIRS=${gtk3}/share:${webkitgtk_4_1}/share:''${XDG_DATA_DIRS:-/usr/share}
  export GSETTINGS_SCHEMA_DIR=${gtk3}/share/gsettings-schemas/${gtk3.name}
  exec ${python}/bin/python3 ${../programs/login/slippi-login.py} "$@"
''
