sha256() {
  cat "${ROOT}"/dependency/sbt-*.tgz.sha256 | cut -f 1 -d ' '
}

uri() {
  echo "https://github.com/sbt/sbt/releases/download/v$(cat "${ROOT}"/dependency/version)/sbt-$(cat "${ROOT}"/dependency/version).tgz"
}
