$.ajaxTransport("+binary", function(options, originalOptions, jqXHR){
    // check for conditions and support for blob / arraybuffer response type
    if (window.FormData && ((options.dataType && (options.dataType == 'binary')) || (options.data && ((window.ArrayBuffer && options.data instanceof ArrayBuffer) || (window.Blob && options.data instanceof Blob)))))
    {
        return {
            // create new XMLHttpRequest
            send: function(_, callback){
				// setup all variables
                var xhr = new XMLHttpRequest(),
                    url = options.url,
                    type = options.type,
		    		// blob or arraybuffer. Default is blob
                    dataType = options.responseType || "blob",
                    data = options.data || null;

                xhr.addEventListener('load', function(){
                    var data = {};
                    data[options.dataType] = xhr.response;
		    		// make callback and send data
                    callback(xhr.status, xhr.statusText, data, xhr.getAllResponseHeaders());
                });

                xhr.open(type, url, true);
                xhr.responseType = dataType;
                xhr.send(data);
            },
            abort: function(){
                jqXHR.abort();
            }
        };
    }
});

var App = React.createClass({
    getInitialState: function() {
        return {modal : "none", path : ""};
    },
    showModal: function(type) {
        this.setState({modal : type})
    },
    hideModal: function() {
        this.setState({modal : "none"})
    },
    updateElements: function() {
        this.refs.elements.update();
    },
    showGallery: function(path) {
        this.setState({path : path, modal : "gallery"});
    },
    render: function() {
        return (
            <div>
                <Menu showModal={this.showModal} />
                <ElementArray ref="elements"  showGallery={this.showGallery} />
                {
                  (this.state.modal === "addStream" ?
                    <AddStreamModal hideModal={this.hideModal} update={this.updateElements} />
                  : (this.state.modal === "uploadFile" ?
                      <UploadFileModal hideModal={this.hideModal} update={this.updateElements} />
                    : (this.state.modal === "gallery" ?
                        <GalleryModal hideModal={this.hideModal} path={this.state.path} />
                      :   <span />
                      )
                    )
                  )
                }
            </div>
        );
    }
});

var Menu = React.createClass({
    addStreamPopup: function() {
        this.props.showModal("addStream")
    },
    uploadFilePopup: function() {
        this.props.showModal("uploadFile")
    },
    render: function() {
        return (
            <div id="upper-title">
                <a href="/interface/" className="brand">DASHME</a>
                <div>
                    <input type="button" value="Add Stream" onClick={this.addStreamPopup} />
                    <input type="button" value="Upload File" onClick={this.uploadFilePopup} />
                </div>
            </div>
        );
    }
});

var AddStreamModal = React.createClass({
    getInitialState: function() {
        return {nameError: false, urlError: false};
    },
    retrieveInput: function () {
        var name = this.refs.name.getDOMNode().value;
        var path = this.refs.url.getDOMNode().value;
        var res = {};
        if (!name || name === "") {
            this.setState({nameError: true})
            res = null;
        }
        if (!path || path === "") {
            this.setState({urlError: true})
            res = null;
        }
        if (res) {
            res.Name = name;
            res.Path = path;
            res.IsLive = this.refs.live.getDOMNode().checked;
            res.Proto = $(this.refs.type.getDOMNode()).find("option:selected").val();
        }
        return res;
    },
    cancel : function() {
        this.props.hideModal();
    },
    validate: function() {
        var data = this.retrieveInput();
        if (!data)
            return;
        $.ajax({
            type : "POST",
            url: "/files",
            data : JSON.stringify(data),
            success: function() {
                this.props.update()
            }.bind(this),
            error: function(xhr, status, err) {
                console.error(this.props.url, status, err.toString());
            }.bind(this)
        });
        this.props.hideModal();
    },
    render: function() {
        return (
            <Modal title="Add stream" validate={this.validate} cancel={this.cancel}>
                <div className="form-line">
                    <label>Name</label>
                    <input type="text" ref="name" className={this.state.nameError ? "error" : ""} />
                </div>
                <div className="form-line">
                    <label>URL</label>
                    <input type="text" ref="url" className={this.state.urlError ? "error" : ""} />
                </div>
                <div className="form-line">
                    <label>Type</label>
                    <select ref="type">
                        <option value="dash">DASH</option>
                        <option value="smooth">Smooth streaming</option>
                    </select>
                </div>
                <div className="form-line">
                    <label>Live</label>
                    <input type="checkbox" ref="live" />
                </div>
            </Modal>
        );
    }
});

var UploadFileModal = React.createClass({
    cancel : function() {
        this.props.hideModal();
    },
    validate: function() {
        var files = this.refs.file.getDOMNode().files;
        var form = new FormData();
        form.append("video", files[0]);
        $.ajax({
            type : "POST",
            url: "/files/upload",
            cache: false,
            data : form,
            processData: false,
            contentType: false,
            success: function() {
                this.props.update()
            }.bind(this),
            error: function(xhr, status, err) {
                console.error(this.props.url, status, err.toString());
            }.bind(this)
        });
        this.props.hideModal();
    },
    render: function() {
        return (
            <Modal title="Add stream" validate={this.validate} cancel={this.cancel}>
                <div className="form-line">
                    <label>File</label>
                    <input type="file" ref="file" name="video" />
                </div>
            </Modal>
        );
    }
});

function extractImages(buffer) {
    var res = [];
    var pos = 0;
    while (pos < buffer.length)
    {
        var length = (buffer[pos] << 24)
            + (buffer[pos + 1] << 16)
            + (buffer[pos + 2] << 8)
            + buffer[pos + 3];
        var ts = (buffer[pos + 4] << 56)
            + (buffer[pos + 5] << 48)
            + (buffer[pos + 6] << 40)
            + (buffer[pos + 7] << 32)
            + (buffer[pos + 8] << 24)
            + (buffer[pos + 9] << 16)
            + (buffer[pos + 10] << 8)
            + buffer[pos + 11];
        var duration = (buffer[pos + 12] << 24)
            + (buffer[pos + 13] << 16)
            + (buffer[pos + 14] << 8)
            + buffer[pos + 15];
        res.push({
            ts : ts,
            duration : duration,
            type : "jpeg",
            data : btoa(String.fromCharCode.apply(null, buffer.subarray(pos + 20, pos + length)))
        });
        pos += length;
    }
    return res;
}

var GalleryModal = React.createClass({
    getInitialState: function() {
        return {images : null, current : 0};
    },
    cancel : function() {
        this.props.hideModal();
    },
    componentDidMount: function() {
        $.ajax({
            url: this.props.path,
            type : "GET",
            dataType : "binary",
            processData : false,
            responseType : "arraybuffer",
            success: function(raw) {
                this.setState({images: extractImages(new Uint8Array(raw))});
            }.bind(this),
            error: function(xhr, status, err) {
                console.error(this.props.url, status, err.toString());
            }.bind(this)
        });
    },
    previousImage : function() {
        if (this.state.current == 0)
            this.setState({current : this.state.images.length - 1});
        else
            this.setState({current : this.state.current - 1});
    },
    nextImage : function() {
        if (this.state.current == this.state.images.length - 1)
            this.setState({current : 0});
        else
            this.setState({current : this.state.current + 1});
    },
    render: function() {
        return (
            <Modal title="Gallery" cancel={this.cancel} >
              <div className="gallery-modal">
                {
                    this.state.images && this.state.images.length > 0 ?
                        <div>
                          <input type="button" value="<<" onClick={this.previousImage} />
                          <img src={"data:image/" + this.state.images[this.state.current].type + ";base64," + this.state.images[this.state.current].data} />
                          <input type="button" value=">>" onClick={this.nextImage} />
                        </div>
                    : ""
                }
              </div>
            </Modal>
        );
    }
});

var Modal = React.createClass({
    cancel : function() {
        if (this.props.cancel)
            this.props.cancel();
    },
    validate : function() {
        if (this.props.validate)
            this.props.validate();
    },
    render: function() {
        return (
            <div>
                <div id="modal-background"></div>
                <div id="modal">
                    <div className="base">
                        <div className="title">{this.props.title}</div>
                        <div className="content">{this.props.children}</div>
                        <div className="footer">
                            { this.props.validate ? <input type="button" value="Save" onClick={this.validate} /> : "" }
                            <input type="button" value={this.props.validate ? "Cancel" : "Close"} onClick={this.cancel} />
                        </div>
                    </div>
                </div>
            </div>
        );
    }
});

var ElementArray = React.createClass({
    getInitialState: function() {
        return {data: []};
    },
    loadFromServer: function() {
        $.ajax({
            url: "/files",
            dataType: "json",
            success: function(data) {
                this.setState({data: data});
            }.bind(this),
            error: function(xhr, status, err) {
                console.error(this.props.url, status, err.toString());
            }.bind(this)
        });
    },
    componentDidMount: function() {
        this.loadFromServer()
    },
    update: function() {
        this.loadFromServer()
    },
    render: function() {
        var elements = this.state.data.map(function (elm) {
            return (
                <Element name={elm.Name} type={elm.Proto} path={elm.Path} live={elm.IsLive} state={elm.State} onUpdate={this.update} showGallery={this.props.showGallery} />
            );
        }.bind(this));
        return (
            <table className="element-list">
                <thead>
                    <tr>
                        <th>Name</th>
                        <th>Type</th>
                        <th>Live</th>
                        <th>Status</th>
                    </tr>
                </thead>
                <tbody>
                    {elements}
                </tbody>
            </table>
        );
    }
});

var Element = React.createClass({
    render: function() {
        return (
            <tr>
                <td>{this.props.name}</td>
                <td>{this.props.type.toUpperCase()}</td>
                <td>{this.props.live ? "Yes" : "No"}</td>
                <ElementGeneration state={this.props.state} live={this.props.live} name={this.props.name} onUpdate={this.props.onUpdate} showGallery={this.props.showGallery} />
            </tr>
        );
    }
});

var ElementGeneration = React.createClass({
    stop : function() {
        $.ajax({
            type : "DELETE",
            url: "/dash/" + this.props.name + "/generate",
            success: function() {
                this.props.onUpdate();
            }.bind(this),
            error: function(xhr, status, err) {
                console.error(this.props.url, status, err.toString());
            }.bind(this)
        });
        this.props.onUpdate();
    },
    generate : function() {
        $.ajax({
            type : "POST",
            url: "/dash/" + this.props.name + "/generate",
            success: function() {
                this.props.onUpdate();
            }.bind(this),
            error: function(xhr, status, err) {
                console.error(this.props.url, status, err.toString());
            }.bind(this)
        });
        this.props.onUpdate();
    },
    showGallery : function() {
        $.ajax({
            type : "GET",
            url : "/dash/" + this.props.name + "/manifest.mpd",
            success : function (data) {
                var chunk_name = $(data).find("AdaptationSet[mimeType='image/jpeg']").find("SegmentTemplate").attr("media");
                this.props.showGallery("/dash/" + this.props.name + "/" + chunk_name);
            }.bind(this),
            error: function(xhr, status, err) {
                console.error(this.props.url, status, err.toString());
            }.bind(this)
        });
    },
    render: function() {
        if (this.props.state == "generated") {
            if (this.props.live) {
                return (
                        <td>
                            <input type="button" value="Stop" onClick={this.stop} />
                        </td>
                );
            } else {
                return (
                        <td>
                            <input type="button" value="Gallery" onClick={this.showGallery} />
                        </td>
                );
            }
        } else if (this.props.state == "generation") {
            return (
                <td className="generation">Generation...</td>
            );
        }  else {
            return (
                <td>
                    <input type="button" value="Generate" onClick={this.generate} />
                </td>
            );
        }
    }
});

React.render(
  <App />,
  document.getElementById("app")
);
