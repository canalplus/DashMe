var App = React.createClass({
    getInitialState: function() {
        return {modal : "none"};
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
    render: function() {
        return (
            <div>
                <Menu showModal={this.showModal} />
                <ElementArray ref="elements" />
                { (this.state.modal === "addStream" ? <AddStreamModal hideModal={this.hideModal} update={this.updateElements} /> : (this.state.modal === "uploadFile" ? <UploadFileModal hideModal={this.hideModal} update={this.updateElements} /> : <span />)) }
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
                            <input type="button" value="Save" onClick={this.validate} />
                            <input type="button" value="Cancel" onClick={this.cancel} />
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
                <Element name={elm.Name} type={elm.Proto} path={elm.Path} live={elm.IsLive} state={elm.State} onUpdate={this.update} />
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
                <ElementGeneration state={this.props.state} live={this.props.live} name={this.props.name} onUpdate={this.props.onUpdate} />
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
                        <td className="generated">Generated</td>
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
