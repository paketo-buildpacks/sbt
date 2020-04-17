sha256() {
  cat "${ROOT}"/dependency/sbt-*.tgz.sha256 | cut -f 1 -d ' '
}
