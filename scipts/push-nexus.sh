for f in $(find linux-64 -type f); do
  curl -T $f -u $1:$2 https://nexus.devops.geointservices.io/content/repositories/beachfront-conda/$f
done
