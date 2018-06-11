toKeep=$(ls ~/miniconda2/conda-bld/linux-64)
echo $toKeep
mkdir linux-64 && cd linux-64
wget -q -r -l1 -e robots=off -nH -nd --reject="index.html*" --no-parent --no-cookies https://nexus.devops.geointservices.io/content/repositories/beachfront-conda/linux-64/ --user=proxy --password=proxy
mv ~/miniconda2/conda-bld/linux-64/* .
conda index .
deleteString="find . -type f"
for f in $toKeep; do
  deleteString=$deleteString" ! -name $f"
done
deleteString=$deleteString" -delete"
$deleteString
ls
