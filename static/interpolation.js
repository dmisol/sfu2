
function triSign(triangle){

    const [x1, y1] = triangle[0];
    const [x2, y2] = triangle[1];
    const [x3, y3] = triangle[2];
    return (x1-x3)*(y2-y3) - (x2-x3)*(y1-y3);
}

function checkTriangle(point,triangle){
    const d1 = triSign([point,triangle[0],triangle[1]]);
    const d2 = triSign([point,triangle[1],triangle[2]]);
    const d3 = triSign([point,triangle[2],triangle[0]]);
    const has_neg = (d1<0) || (d2<0) || (d3<0);
    const has_pos = (d1>0) || (d2>0) || (d3>0);
    return !(has_neg && has_pos);

}

function findTriangleContaining(point,triangles){
    var ret = -1;
    for (let j = 0; j < triangles.length; j++) {
            const result = checkTriangle(point,triangles[j]);
            if (result) {
                ret = j;
                break;
            }
//            console.log(result);
    }
    return ret;
}

function vLength(v){
    return Math.sqrt(v[0] * v[0] + v[1] * v[1]);
}
function vLength2(v){
    return v[0] * v[0] + v[1] * v[1];
}
function dot(v1,v2){
    return v1[0] * v2[0] + v1[1] * v2[1];
}

function vSub(v1,v2){
    return [v1[0]-v2[0],v1[1]-v2[1]];

}

function distAndWeight(v0,v1){
    const v1Len = vLength(v1) + 0.0001;
    var proj = dot(v0,v1)/v1Len;
    const dist = Math.sqrt(vLength2(v0) - proj * proj);
    const weight = proj/v1Len;
    return [dist,weight];
}

function vNorm(v){
    const sum = v.reduce((accumulator, currentValue) => accumulator + currentValue, 0);
    const ret = [];
    for (const vv of v) {
        ret.push(vv/sum);
    }
    return ret;
}

function findWeightsForPoly(poly,point){
    const dists = [];
    const weights = [];
    for (let i = 0; i < poly.length - 1; i++) {
        const v1 = vSub(poly[i+1],poly[i]);
        const v0 = vSub(point,poly[i]);
        const [dist,weight] = distAndWeight(v0,v1);
        dists.push(1.0/(dist+0.001));
        weights.push(weight);
    }
    const v1 = vSub(poly[0],poly[poly.length-1]);
    const v0 = vSub(point,poly[poly.length-1]);
    const [dist,weight] = distAndWeight(v0,v1);
    dists.push(1.0/(dist+0.001));
    weights.push(weight);
    const distN = vNorm(dists);

    const pointWeights = [0.0,0.0,0.0];
    for (let i = 0; i < poly.length - 1; i++) {
        pointWeights[i] += (1 - weights[i]) * distN[i];
        pointWeights[i+1] += weights[i] * distN[i];
    }
    pointWeights[poly.length-1] += (1 - weights[poly.length-1]) * distN[poly.length-1];
    pointWeights[0] += weights[poly.length-1] * distN[poly.length-1];
    return vNorm(pointWeights);

}
function makeInterUnit(point,triangles,indices,border){
    var triIdx = findTriangleContaining(point,triangles);
    var calculatedPoint = point;
    if (triIdx == -1) {
//        console.log("borderPoint");
//        const borderPoint = findIntersection([[0.0,0.0],[1.0,0.1]],[[0.5,1.0],[0.55,-1.0]]);
//        console.log(borderPoint);
        var borderPoint = null;
        for (const b of border) {
            borderPoint = findIntersection(b,[point,[0.5,0.45]]);
//            console.log(borderPoint);
            if (borderPoint!=null){
                break;
            }
        }
         if (borderPoint==null){
            return null;
         }
         calculatedPoint = pointBetweenPoints([0.5,0.45],borderPoint,0.98);
         triIdx = findTriangleContaining(calculatedPoint,triangles);
    }
//    console.log("calculatedPoint");
//    console.log(calculatedPoint);
    const weights = findWeightsForPoly(triangles[triIdx],calculatedPoint);

    const bshpIdx = [];
    for (let i = 0; i < 3; i++) {
        bshpIdx.push(indices[triIdx * 3 + i]);
    }
    return [bshpIdx,weights,[point[0]-calculatedPoint[0],point[1]-calculatedPoint[1]]]

}

function pointBetweenPoints(p1,p2,percent){
    const [x1,y1] = p1;
    const [x2,y2] = p2;
    const x = x1 + (x2-x1) * percent;
    const y = y1 + (y2-y1) * percent;
    return [x,y];
}

function rCont(range,val){
    return range[0]<=val &&  range[1]>=val;
}

function findIntersection(line1,line2){
    const [s1,e1] = line1;
    const [s2,e2] = line2;
    const [s1x,s1y] = s1;
    const [e1x,e1y] = e1;
    const [s2x,s2y] = s2;
    const [e2x,e2y] = e2;
    const slope1 = (e1y-s1y)/(e1x-s1x);
    const yIntercept1 = s1y - slope1 * s1x;
//    console.log([slope1,yIntercept1]);

    const slope2= (e2y-s2y)/(e2x-s2x);
    const yIntercept2 = s2y - slope2 * s2x;
//     console.log([slope2,yIntercept2]);
    if (slope1 == slope2) {
        return null;
    }
    const x = (yIntercept2 - yIntercept1) / (slope1 - slope2);
    const y = slope1 * x + yIntercept1;
//    console.log([x,y]);
    const x1Range = [Math.min(s1x, e1x),Math.max(s1x, e1x)];
    const x2Range = [Math.min(s2x, e2x),Math.max(s2x, e2x)];
    const y1Range = [Math.min(s1y, e1y),Math.max(s1y, e1y)];
    const y2Range = [Math.min(s2y, e2y),Math.max(s2y, e2y)];
    if (rCont(x1Range,x) && rCont(y1Range,y) && rCont(x2Range,x) && rCont(y2Range,y)){
        return [x,y];
    }
    else{
       return null;
    }

}
