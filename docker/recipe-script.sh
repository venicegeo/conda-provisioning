export PATH="$HOME/miniconda2/bin:$PATH"
echo Clearing out conda-bld
rm -rf ~/miniconda2/conda-bld
echo Rebuilding conda-bld
mkdir -p ~/miniconda2/conda-bld/linux-64
mkdir -p ~/miniconda2/conda-bld/noarch
conda index ~/miniconda2/conda-bld/linux-64
conda index ~/miniconda2/conda-bld/noarch
echo Adding channels
conda config --remove channels defaults
conda config --add channels defaults
conda config --add channels conda-forge
conda config --add channels bioconda
conda config --add channels local
cd share/recipes
recipes=$(ls)
for f in $recipes; do
  echo "Starting build for $f"
  conda build $f --old-build-string -q
done
cd
mkdir -p channel/linux-64
mkdir -p channel/noarch
cd channel/linux-64
mv ~/miniconda2/conda-bld/linux-64/* .
conda index .
cd ../noarch
conda index .
cd ../..
mv channel share/
