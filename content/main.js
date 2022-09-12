"use strict";

// -----------------------------------------------

window.FassParamsConf = {
  'pathcmdcontainer' : {
    'title': 'Path of command for executor\'s container',
    'help' : '', 
    'construct': (evt) => {},
    'confirm': (evt) => {}
  }, 
  'domain' : {
    'title': 'Listening domain',
    'help' : '', 
    'construct': (evt) => {},
    'confirm': (evt) => {}
  },
  'authorization' : {
    'title': 'Content of header Authorization',
    'help' : '', 
    'construct': (evt) => {},
    'confirm': (evt) => {}
  },
  'adress' : {
    'title': 'Adress of bind',
    'help' : '"0.0.0.0" for all interfaces', 
    'construct': (evt) => {},
    'confirm': (evt) => {}
  },
  'listen' : {
    'title': 'Port of bind',
    'help' : 'Valid range : 1 to 65535', 
    'construct': (evt) => {},
    'confirm': (evt) => {}
  },
  'tls' : {
    'title': 'Tuple of paths for certificat and key TLS',
    'help' : 'Separator between two parts ":"', 
    'construct': (evt) => {},
    'confirm': (evt) => {}
  },
  'delay' : {
    'title': 'Delay',
    'help' : '', 
    'construct': (evt) => {},
    'confirm': (evt) => {}
  },
  'ui' : {
    'title': 'Distant path for UI\'s content',
    'help' : 'Can be relative or absolute root', 
    'construct': (evt) => {},
    'confirm': (evt) => {}
  },
  'tmp' : {
    'title': 'Distant path for temporary files',
    'help' : 'Can be relative or absolute root', 
    'construct': (evt) => {},
    'confirm': (evt) => {}
  },
  'prefix' : {
    'title': 'Prefix for URI',
    'help' : 'Must be a valid string', 
    'construct': (evt) => {},
    'confirm': (evt) => {}
  }
};

// -----------------------------------------------

class FaassConfiguration {
  static apiInternalPath = '/api/configuration'; 
  headers = new Headers();
  distantConf = null; 
  inProgress = false; 
  constructor( password ) {}
  refresh() { 
    this.inProgress = true; 
    return this.#getWebRequest()
      .then( 
        c => {
          this.distantConf = c; 
          this.inProgress = false; 
          return Promise.resolve(); 
        }
      )
      .catch( 
        e => {
          this.inProgress = false; 
          if ( typeof e == "number" ) {
            console.error( `err refresh, status code : ${e}` ); 
            return Promise.reject( `invalid status code (${e})` ); 
          } else { 
            console.error( `err refresh, internal error : ${e}` ); 
            return Promise.reject( `invalid response body` ); 
          }
        }
      )
    ;
  }
  #getWebRequest() {
    this.headers.set( 
      'Authorization', 
      document
        .getElementById( 'auth_form_conf' )
        .value 
    ); 
    const request = new Request( 
    window.location.origin+FaassConfiguration.apiInternalPath, 
      {
        method: 'GET', 
        headers: this.headers 
      }
    );
    return fetch( request )
      .then( 
        r => {
          if (r.status === 200) {
            return r.json(); 
          } else {
            return Promise.reject( r.status );
          }
        }
      )
  }
  formConf( wantFragment ) {
    if ( this.distantConf == null ) {
      return goTo( 
        'error', 
        'the local conf form is requested but has a null value' 
      ); 
    }
    var f = ''; 
    const contexte = 'type="conf" action="update"'; 
    for ( const [ key, value ] of Object.entries( this.distantConf ) ) {
      if ( key == 'routes' ) 
        continue; 
      f += `
        <input-${typeof value} ${contexte} name="${key}">${value}</input-${typeof value}>
      `;
    }
    const body = `
      <form-object ${contexte}>
        ${f}
        <input type="submit" value="Update" />
      </form-object> 
    `; 
    if ( wantFragment == true ) {
      const fragment = document.createDocumentFragment();
      fragment.innerHTML = body; 
      return fragment; 
    } 
    return body;
  }
}

// -----------------------------------------------

class TemplateGeneric extends HTMLElement {
  constructor() {
    super();
    this.idTemplate = Object.getPrototypeOf( this )
      .constructor
      .idTemplate; 
    const template = this.deal( 
      document.getElementById( this.idTemplate )
        .content
        .cloneNode(true)
    );  
    const shadowRoot = this.attachShadow( {mode: 'open'} )
      .appendChild( template );
  } 
  deal( template ) {
    return template; 
  }
}

// -----------------------------------------------

class TemplateConfGet extends TemplateGeneric { 
  static idTemplate = 't_conf_get'; 
  deal( template ) {
    if ( this.attributes.getNamedItem( 'last' ) != null ) {
      var a = this.attributes.removeNamedItem( 'last' ); 
      template.querySelector('input').attributes.setNamedItem( a ); 
    } 
    template.querySelector('input').addEventListener(
      "click", 
      evt => goTo( evt.target.getAttribute( 'last' ) )
    )
    return template;
  }
}
customElements.define( 'conf-get', TemplateConfGet );

// -----------------------------------------------

class TemplateError extends TemplateGeneric { 
  static idTemplate = 't_error'; 
  deal( template ) {
    var m = 'an unexpected error was found';
    if ( this.attributes.getNamedItem( 'detail' ) != null ) {
      m = this.attributes.getNamedItem( 'detail' ).value; 
    } 
    template.querySelector('.detail').innerHTML = m; 
    return template;
  }
}
customElements.define( 'error-detail', TemplateError );

// -----------------------------------------------

class TemplateInput extends TemplateGeneric { 
  deal( template ) {
    var relativeName = this.attributes
      .getNamedItem( 'name' )
      .value;
    if ( window.FassParamsConf[relativeName] == undefined ) {
      return goTo( 
        'error', 
        `the key '${relativeName}' in distant conf was not found in local params` 
      );
    }
    var relativeId = this.attributes
      .getNamedItem( 'type' )
      .value + '-' + relativeName; 
    var elInput = template.querySelector( 'input' ); 
    elInput.attributes
      .getNamedItem( 'name' )
      .value = relativeName;
    elInput.attributes
      .getNamedItem( 'id' )
      .value = relativeId;
    var elLabel = template.querySelector( 'label' ); 
    elLabel.attributes
      .getNamedItem( 'for' )
      .value = relativeId;
    elLabel.innerText = relativeName; 
    var elLegend = template.querySelector( 'legend' ); 
    elLegend.innerText = window.FassParamsConf[relativeName]['title']; 
    if ( window.FassParamsConf[relativeName]['help'] != "" ) {
      template.querySelector( 'span' ).innerText = window.FassParamsConf[relativeName]['help']; 
    }
    return template;
  }
}

class TemplateInputString extends TemplateInput { 
  static idTemplate = 't_form_input'; 
  deal( template ) {
    template = super.deal( template ); 
    template.querySelector( 'input' ).type = 'string';
    template.querySelector( 'input' ).value = this.innerText;
    return template;
  }
}
customElements.define( 'input-string', TemplateInputString );

class TemplateInputNumber extends TemplateInput { 
  static idTemplate = 't_form_input'; 
  deal( template ) {
    template = super.deal( template ); 
    template.querySelector( 'input' ).type = 'number';
    template.querySelector( 'input' ).value = parseInt( this.innerText );
    return template;
  }
}
customElements.define( 'input-number', TemplateInputNumber );

// -----------------------------------------------

function goTo( part, ...rest ) {
  switch ( part ) {
    case 'error': 
      document.getElementById('content').innerHTML = `
        <error-detail detail="${rest[0]}"></error-detail>
      `;
      break 
    case 'before-refresh':
      document.getElementById('content').innerHTML = `
        <conf-get last="refresh"></conf-get>
      `;
      break
    case 'refresh': 
      document.getElementById('content').innerHTML = 'wait...'; 
      window.FaassConfiguration.refresh()
        .then( 
          _ => {
            document.getElementById('content').innerHTML = `
              <h1>Edit global configuration</h1>
              ${window.FaassConfiguration.formConf( false )}
            `; 
          }
        )
        .catch( 
          e => document.getElementById('content').innerHTML = `
            <error-detail detail="${e}"></error-detail>
            <conf-get last="refresh"></conf-get>
          `
        );
      break
  }
}

// -----------------------------------------------

window.addEventListener( 
  "load", 
  () => {
    window.FaassConfiguration = new FaassConfiguration();
    document.getElementById( 'link_to_edit_conf' ).addEventListener( 
      'click',
      evt => goTo( 'refresh' )
    ); 
    document.getElementById( 'link_to_edit_routes' ).addEventListener( 
      'click',
      evt => goTo( 'error', 'not implemented' )
    ); 
    document.getElementById( 'link_to_request' ).addEventListener( 
      'click',
      evt => goTo( 'error', 'not implemented' )
    ); 
    goTo( 'before-refresh' ); 
  }
);




