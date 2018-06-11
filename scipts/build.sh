export PATH=$PATH:$HOME/miniconda2/bin
echo Clearing out conda-build
rm -rf ~/miniconda2/conda-bld
mkdir -p ~/miniconda2/conda-bld/linux-64
mkdir -p ~/miniconda2/conda-bld/noarch
conda index ~/miniconda2/conda-bld/linux-64
conda index ~/miniconda2/conda-bld/noarch
echo Ordering channels
conda config --add channels defaults
conda config --add channels bioconda
conda config --add channels conda-forge
conda config --add channels local
cd recipes
recipeFolders=$(ls)
for f in $recipeFolders; do
  echo "Starting build for $f"
  conda build $f --old-build-string -q
done
