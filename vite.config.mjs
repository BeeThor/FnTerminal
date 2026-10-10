import {defineConfig} from 'vite'
import vue from '@vitejs/plugin-vue'
export default defineConfig({root:'frontend',base:'/app/fnterminal/',plugins:[vue()],build:{outDir:'../.build/web',emptyOutDir:true,target:'es2022',sourcemap:false,cssCodeSplit:false},server:{host:'127.0.0.1'}})
