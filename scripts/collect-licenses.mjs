import {mkdirSync,readFileSync,readdirSync,copyFileSync,existsSync,writeFileSync} from 'node:fs';
import {join} from 'node:path';
const target=process.argv[2];mkdirSync(target,{recursive:true});
const lock=JSON.parse(readFileSync('package-lock.json','utf8'));const records=[];
for(const [path,meta] of Object.entries(lock.packages)){
 if(!path||meta.dev||!existsSync(path))continue;
 const pkg=JSON.parse(readFileSync(join(path,'package.json'),'utf8'));
 const name=pkg.name.replaceAll('/','__').replaceAll('@','');
 const dir=join(target,name);mkdirSync(dir,{recursive:true});
 const files=readdirSync(path).filter(file=>/^(license|licence|copying|notice)(\.|$)/i.test(file));
 for(const file of files)copyFileSync(join(path,file),join(dir,file));
 records.push({name:pkg.name,version:pkg.version,license:pkg.license||meta.license,files});
}
writeFileSync(join(target,'INDEX.json'),JSON.stringify(records,null,2)+'\n');
