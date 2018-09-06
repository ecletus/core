'use strict';


var gulp = require('gulp'),
  babel = require('gulp-babel'),
  eslint = require('gulp-eslint'),
  plugins = require('gulp-load-plugins')(),
  plumber = require('gulp-plumber'),
  fs = require('fs'),
  path = require('path'),
  es = require('event-stream'),
  rename = require('gulp-rename'),
  walk = require('fs-walk');

const prefix = '/assets/static/admin/';


function pj() {
  let args = Array.prototype.slice.call(arguments);
  return args.join('/').replace(/\/+/g, '/');
}

var moduleName = (function () {
  var args = process.argv,
    length = args.length,
    i = 0,
    name,
    subName,
    useSubName;

  while (i++ < length) {
    if (/^:+(\w+)/i.test(args[i])) {
      name = args[i].split(':')[1];
      subName = args[i].split(':')[2];
      useSubName = args[i].split(':')[3];
      break;
    }
  }
  return {
    name: name,
    subName: subName,
    useSubName: useSubName
  };
})();

function loadTasks(lib, prefix) {
  prefix = prefix || "";
  walk.walkSync('./gulp_tasks/' + prefix, function(basedir, filename, stat) {
    if (!stat.isDirectory() && /\.js$/.test(filename)) {
      require("./" + basedir + "/" + filename).Task(lib, gulp);
    }
  });
}

// Admin Module
// Command: gulp [task]
// Admin is default task
// Watch Admin module: gulp
// -----------------------------------------------------------------------------

function adminTasks() {
  let pathto = function (file) {
      return pj('..', 'admin', prefix, file);
    },
    scripts = {
      src: pathto('javascripts/app/*.js'),
      dest: pathto('javascripts'),
      qor: pathto('javascripts/qor/*.js'),
      qorInit: pathto('javascripts/qor/qor-config.js'),
      qorCommon: pathto('javascripts/qor/qor-common.js'),
      qorAdmin: [pathto('javascripts/qor.js'), pathto('javascripts/app.js')],
      all: ['gulpfile.js', pathto('javascripts/qor/*.js')]
    },
    styles = {
      src: pathto('stylesheets/scss/{app,qor}.scss'),
      dest: pathto('stylesheets'),
      vendors: pathto('stylesheets/vendors'),
      main: pathto('stylesheets/{qor,app}.css'),
      qorAdmin: [pathto('stylesheets/vendors.css'), pathto('stylesheets/qor.css'), pathto('stylesheets/app.css')],
      scss: pathto('stylesheets/scss/**/*.scss')
    };

  loadTasks({
    scripts: scripts,
    styles: styles,
    prefix: prefix,
    pj: pj,
    pathto: pathto,
    plugins: plugins
  }, "admin");
}

// -----------------------------------------------------------------------------
// Other Modules
// Command: gulp [task] --moduleName--subModuleName
//
//  example:
// Watch Worker module: gulp --worker
//
// if module's assets just as normal path:
// moduleName/assets/static/themes/moduleName/assets/javascripts(stylesheets)
// just use gulp --worker
//
// if module's assets in enterprise as normal path:
// moduleName/assets/static/themes/moduleName/assets/javascripts(stylesheets)
// just use gulp --microsite--enterprise
//
// if module's assets path as Admin module:
// moduleName/assets/static/javascripts(stylesheets)
// you need set subModuleName as admin
// gulp --worker--admin
//
// if you need run task for subModule in modules
// example: worker module inline_edit subModule:
// gulp --worker--inline_edit
//
// gulp --media--media_library--true
//
// -----------------------------------------------------------------------------

function moduleTasks(moduleNames) {
  var moduleName = moduleNames.name,
    subModuleName = moduleNames.subName,
    useSubName = moduleNames.useSubName;

  var pathto = function (file) {
    if (moduleName && subModuleName) {
      if (subModuleName === 'admin') {
        return pj('..', moduleName, '/', prefix, file);
      } else if (useSubName) {
        return pj('..', moduleName, '/', subModuleName, prefix, 'themes', subModuleName, file);
      } else {
        return pj('..', moduleName, '/', subModuleName, prefix, 'themes', moduleName, file);
      }
    }
    return pj('..', moduleName, '/', prefix, 'themes', moduleName, file);
  };

  var scripts = {
    src: pathto('javascripts/'),
    watch: pathto('javascripts/**/*.js')
  };
  var styles = {
    src: pathto('stylesheets/'),
    watch: pathto('stylesheets/**/*.scss')
  };

  function getFolders(dir) {
    return fs.readdirSync(dir).filter(function (file) {
      return fs.statSync(path.join(dir, file)).isDirectory();
    });
  }

  gulp.task('js', function () {
    var scriptPath = scripts.src;
    var folders = getFolders(scriptPath);

    var task = folders.map(function (folder) {
      return gulp
        .src(path.join(scriptPath, folder, '/*.js'))
        .pipe(plumber())
        .pipe(
          eslint({
            configFile: '.eslintrc'
          })
        )
        .pipe(
          babel({
            presets: ['es2015']
          })
        )
        .pipe(eslint.format())
        .pipe(plugins.concat(folder + '.js'))
        .pipe(
          plugins.uglify({
            drop_debugger: false
          })
        )
        .pipe(gulp.dest(scriptPath));
    });

    return es.concat.apply(null, task);
  });

  gulp.task('css', function () {
    var stylePath = styles.src;
    var folders = getFolders(stylePath);
    var task = folders.map(function (folder) {
      return gulp
        .src(path.join(stylePath, folder, '/*.scss'))
        .pipe(plumber())
        .pipe(
          plugins.sass({
            outputStyle: 'compressed'
          })
        )
        .pipe(plugins.minifyCss())
        .pipe(rename(folder + '.css'))
        .pipe(gulp.dest(stylePath));
    });

    return es.concat.apply(null, task);
  });

  gulp.task('watch', function () {
    var moduleScript = gulp.watch(scripts.watch, {debounceDelay: 2000}, ['js']);
    gulp.watch(styles.watch, ['css']);

    moduleScript.on('change', function (event) {
      console.log(':==> File ' + event.path + ' was ' + event.type + ', running tasks...');
    });
  });

  gulp.task('default', ['watch']);
  gulp.task('release', ['js', 'css']);
}

// Init
// -----------------------------------------------------------------------------

if (moduleName.name) {
  var taskPath = pj(moduleName.name, prefix, 'themes', + moduleName.name);
  var runModuleName = 'Running "' + moduleName.name + '" module task in "' + taskPath + '"...';

  if (moduleName.subName) {
    if (moduleName.subName === 'admin') {
      taskPath = pj(moduleName.name, prefix);
      runModuleName = 'Running "' + moduleName.name + '" module task in "' + taskPath + '"...';
    } else if (moduleName.useSubName) {
      taskPath = pj(moduleName.name, moduleName.subName, prefix, 'themes', moduleName.subName);
      runModuleName = 'Running "' + moduleName.name + ' > ' + moduleName.subName + '" module task in "' + taskPath + '"...';
    } else {
      taskPath = pj(moduleName.name, moduleName.subName, prefix, 'themes', moduleName.name);
      runModuleName = 'Running "' + moduleName.name + ' > ' + moduleName.subName + '" module task in "' + taskPath + '"...';
    }
  }
  console.log(runModuleName);
  moduleTasks(moduleName);
} else {
  console.log('Running "admin" module task in "' + pj('..', 'admin', prefix) + '...');
  adminTasks();
}
