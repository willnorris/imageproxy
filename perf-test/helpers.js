
export const getRandomUrl = (randomize) => {
    let path = paths[randomInt(0, paths.length - 1)];

    if (randomize) {
        path = path.replace(/x[0-9]+/, `x${randomInt(100, 1000)}`);
    }
    return  `https://qa.totalwine.com/dynamic${path}`;
}
//https://qa.totalwine.com/dynamic/x220,375ml/sys_master/twmmedia/h04/h19/14405062262814.png

export const randomInt = (min, max) => { // min and max included
    return Math.floor(Math.random() * (max - min + 1) + min)
}

let paths = [
    "/x220,375ml/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h04/h19/14405062262814.png",
    "/x1000,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h43/hc1/14938355531806.png",
    "/270x,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h71/hb7/12267817926686.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hfd/h83/12034978611230.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hb4/h56/9683874873374.png",
    "/x1000,50ml,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h09/hbe/15448923635742.png",
    "/sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hba/h71/9299438862366.png",
    "/x1000,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h95/h0a/15448916230174.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h3a/h99/11192415289374.png",
    "/x220,200ml,webp/https://storage.googleapis.com/hybris-public-production/sys_master/cmsmedia/hf6/h2c/8979036504094.png",
    "/x155,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h02/h5d/15449032687646.png",
    "/x155,4pk,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h36/h62/16964328456222.png",
    "/sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h6b/h86/8805919490078.png",
    "/sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h21/h5f/13763295019038.png",
    "/sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h07/h22/12338782142494.png",
    "/sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h02/h1a/14590886281246.png",
    "/x450,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hf2/he7/26565062557726.png",
    "/x450,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/had/hef/14521135136798.png",
    "/x450,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h91/hf4/8802820423710.png",
    "/x450,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h8a/hb9/13512306458654.png",
    "/x450,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h72/hdd/10777597444126.png",
    "/x450,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h63/h69/13713376870430.png",
    "/x450,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/h48/hec/15280777723934.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hfd/hf2/8810640211998.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hfc/h4b/12245177139230.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hfa/hdf/12056616992798.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hf8/he1/14271263178782.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hf8/h7f/26701864632350.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hf7/hd0/16102248120350.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hf6/h79/16156004220958.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hf5/h7d/17159611777054.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hf4/ha7/8804650123294.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hf2/h60/15992134467614.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hf1/h1d/10675638468638.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hf0/h36/27446335406110.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hed/h69/28405437661214.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hec/hc8/9923797680158.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/heb/h84/13474955788318.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/heb/h80/28335705817118.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hea/h53/27178157375518.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/he7/h70/12322315173918.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/he7/h2d/26523099496478.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/he7/h28/27279495888926.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/he4/he7/14936818090014.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/he1/hd8/10873843417118.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/he1/h9c/14575203713054.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/he1/h35/28358492749854.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/he0/hf0/13909180448798.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/he0/h23/8798265049118.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hdf/hda/14202389987358.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hde/hb7/28597737947166.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hdc/h80/16036737417246.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hdb/h8f/26799410315294.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hd9/h2f/11941568741406.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hd5/h4f/16872939192350.png",
    "/x220,sq,webp/https://storage.googleapis.com/hybris-public-production/sys_master/twmmedia/hd4/h59/28134069075998.png"
]
//
//
// paths = [
//     'dynamic/x250,sq,k2{rand}/media/sys_master/twmmedia/h39/hc0/15078186647582.png',
// ];
